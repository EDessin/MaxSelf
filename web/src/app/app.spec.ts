import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { TestBed } from '@angular/core/testing';
import Aura from '@primeuix/themes/aura';
import { providePrimeNG } from 'primeng/config';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { App } from './app';

const rule = {
  type: 'exercise',
  title: 'Move Your Body',
  xp: 40,
  stat: 'strength',
  icon: 'dumbbell',
  color: '#ff5a5f'
};

function dashboard(totalXp = 0) {
  return {
    user: {
      id: 'user-1',
      email: 'demo@maxself.app',
      displayName: 'Max Player'
    },
    progress: {
      level: 1,
      totalXp,
      currentLevelXp: totalXp,
      nextLevelXp: 100,
      streakDays: totalXp > 0 ? 1 : 0,
      stats: {
        strength: totalXp,
        fuel: 0,
        recovery: 0,
        mindset: 0,
        consistency: 0
      }
    },
    activities: [],
    rules: [rule]
  };
}

describe('App', () => {
  let http: HttpTestingController | undefined;
  let storage: Record<string, string>;

  beforeEach(async () => {
    storage = {};
    vi.stubGlobal('localStorage', {
      getItem: vi.fn((key: string) => storage[key] ?? null),
      setItem: vi.fn((key: string, value: string) => {
        storage[key] = value;
      }),
      removeItem: vi.fn((key: string) => {
        delete storage[key];
      }),
      clear: vi.fn(() => {
        storage = {};
      })
    });

    await TestBed.configureTestingModule({
      imports: [App],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        provideNoopAnimations(),
        providePrimeNG({
          theme: {
            preset: Aura,
            options: {
              darkModeSelector: '.dark-mode'
            }
          }
        })
      ]
    }).compileComponents();

    http = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    http?.verify();
    http = undefined;
    vi.unstubAllGlobals();
  });

  it('should create the app', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;

    expect(app).toBeTruthy();
  });

  it('should render the MaxSelf title', async () => {
    const fixture = TestBed.createComponent(App);
    await fixture.whenStable();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('h1')?.textContent).toContain('MaxSelf');
  });

  it('should close the activity dialog after a successful XP claim', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.token = 'token';
    app.dashboard.set(dashboard());
    app.openActivity(rule);

    app.saveActivity();

    const request = http!.expectOne('http://localhost:8080/api/activities');
    expect(request.request.method).toBe('POST');
    expect(app.activitySaving()).toBe(true);

    request.flush(dashboard(40));

    expect(app.activitySaving()).toBe(false);
    expect(app.activityDialogOpen()).toBe(false);
    expect(app.selectedRule()).toBeUndefined();
    expect(app.dashboard()?.progress.totalXp).toBe(40);
  });

  it('should remove the rendered activity modal after clicking Claim XP successfully', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.dashboard.set(dashboard());
    app.openActivity(rule);
    fixture.detectChanges();
    app.token = 'token';

    const root = fixture.nativeElement as HTMLElement;
    expect(root.querySelector('.activity-dialog')).not.toBeNull();

    const claimButton = Array.from(root.querySelectorAll('button'))
      .find((button) => button.textContent?.includes('Claim XP'));
    claimButton?.click();

    const request = http!.expectOne('http://localhost:8080/api/activities');
    request.flush(dashboard(40));
    await fixture.whenStable();

    expect(root.querySelector('.modal-backdrop')).toBeNull();
    expect(root.querySelector('.activity-dialog')).toBeNull();
  });

  it('should leave the login button loading state after authentication completes', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const loginButton = Array.from(root.querySelectorAll('button'))
      .find((button) => button.textContent?.includes('Enter MaxSelf'));

    loginButton?.click();

    const loginRequest = http!.expectOne('http://localhost:8080/api/auth/login');
    expect(loginRequest.request.method).toBe('POST');
    loginRequest.flush({ token: 'token' });

    const dashboardRequest = http!.expectOne('http://localhost:8080/api/dashboard');
    expect(dashboardRequest.request.method).toBe('GET');
    dashboardRequest.flush(dashboard(40));
    await fixture.whenStable();

    expect(root.querySelector('.button-spinner')).toBeNull();
    expect(root.textContent).toContain('Max Player');
    expect(root.textContent).toContain('40 total XP');
  });
});
