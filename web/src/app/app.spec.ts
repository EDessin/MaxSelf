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
  title: 'Strength Session',
  xp: 40,
  stat: 'strength',
  icon: 'dumbbell',
  color: '#ff5a5f'
};

const allRules = [
  { type: 'cardio', title: 'Cardio Session', xp: 30, stat: 'cardio', icon: 'flame', color: '#f59e0b' },
  rule,
  { type: 'healthy_meal', title: 'Nourishing Meal', xp: 25, stat: 'fuel', icon: 'apple', color: '#22c55e' },
  { type: 'hydration', title: 'Hydration Boost', xp: 10, stat: 'fuel', icon: 'droplet', color: '#38bdf8' },
  { type: 'sleep', title: 'Sleep Goal Met', xp: 35, stat: 'recovery', icon: 'moon', color: '#6366f1' },
  { type: 'mindfulness', title: 'Mindset Moment', xp: 20, stat: 'mindset', icon: 'sparkles', color: '#a855f7' },
  { type: 'recovery', title: 'Recovery Ritual', xp: 20, stat: 'recovery', icon: 'heart-pulse', color: '#14b8a6' },
  { type: 'bonus', title: 'Bonus Quest', xp: 5, stat: 'mindset', icon: 'unknown', color: '#f59e0b' }
];

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
        cardio: 0,
        strength: totalXp,
        fuel: 0,
        recovery: 0,
        mindset: 0,
        cardio_consistency: 0,
        strength_consistency: 0,
        fuel_consistency: 0,
        recovery_consistency: 0,
        mindset_consistency: 0
      }
    },
    activities: [],
    rules: [rule]
  };
}

function fullDashboard() {
  const now = new Date().toISOString();
  return {
    ...dashboard(140),
    progress: {
      ...dashboard(140).progress,
      level: 2,
      currentLevelXp: 40,
      nextLevelXp: 200,
      streakDays: 3,
      stats: {
        cardio: 30,
        strength: 40,
        fuel: 35,
        recovery: 55,
        mindset: 25,
        cardio_consistency: 5,
        strength_consistency: 10,
        fuel_consistency: 15,
        recovery_consistency: 20,
        mindset_consistency: 25,
        consistency: 99
      }
    },
    activities: allRules.map((activityRule, index) => ({
      id: `activity-${index}`,
      type: activityRule.type,
      title: activityRule.title,
      notes: '',
      xp: activityRule.xp,
      stat: activityRule.stat,
      occurredAt: now
    })),
    rules: allRules
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

  it('should render the dashboard, quick actions, stats, and recent wins', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.dashboard.set(fullDashboard());
    fixture.detectChanges();
    await fixture.whenStable();

    const root = fixture.nativeElement as HTMLElement;
    expect(root.textContent).toContain('Max Player');
    expect(root.textContent).toContain('Level');
    expect(root.textContent).toContain('140 total XP');
    expect(root.textContent).toContain('Core Stats');
    expect(root.textContent).toContain('Cardio');
    expect(root.textContent).toContain('Strength');
    expect(root.textContent).toContain('Fuel');
    expect(root.textContent).toContain('5 consistency XP');
    expect(root.textContent).toContain('10 consistency XP');
    expect(root.textContent).not.toContain('99 XP');
    expect(root.textContent).toContain('Recent Wins');
    expect(Array.from(root.querySelectorAll('.quest-column-header span')).map((header) => header.textContent?.trim()))
      .toEqual(['Cardio', 'Strength', 'Fuel', 'Recovery', 'Mindset']);
    expect(root.querySelectorAll('.action-tile').length).toBe(allRules.length);
    expect(root.querySelectorAll('.quest-column').length).toBe(5);
    expect(root.querySelectorAll('tbody tr').length).toBe(allRules.length);
    expect(app.progressPercent()).toBe(20);
    expect(app.todayXp()).toBe(allRules.reduce((sum, activityRule) => sum + activityRule.xp, 0));
    expect(app.iconFor('missing')).toBe('star');
    expect(app.colorFor('missing')).toBe('#f59e0b');
  });

  it('should open and close the activity dialog from a dashboard action', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.dashboard.set(fullDashboard());
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const action = Array.from(root.querySelectorAll<HTMLButtonElement>('.action-tile'))
      .find((button) => button.textContent?.includes('Hydration Boost'));
    action?.click();
    await fixture.whenStable();

    expect(root.querySelector('.activity-dialog')).not.toBeNull();
    expect(root.textContent).toContain('+10 Health XP');

    const closeButton = root.querySelector('button[aria-label="Close activity dialog"]') as HTMLButtonElement;
    closeButton.click();
    await fixture.whenStable();

    expect(root.querySelector('.activity-dialog')).toBeNull();
    expect(app.selectedRule()).toBeUndefined();
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

  it('should show an auth error and clear pending state when login fails', async () => {
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const loginButton = Array.from(root.querySelectorAll('button'))
      .find((button) => button.textContent?.includes('Enter MaxSelf'));
    loginButton?.click();

    const loginRequest = http!.expectOne('http://localhost:8080/api/auth/login');
    loginRequest.flush({ error: 'nope' }, { status: 401, statusText: 'Unauthorized' });
    await fixture.whenStable();

    expect(root.querySelector('.button-spinner')).toBeNull();
    expect(root.textContent).toContain('Could not log in');
    expect(storage['maxself.token']).toBeUndefined();
  });

  it('should register, load the dashboard, and use register-specific errors', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.authMode = 'register';
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    expect(root.textContent).toContain('Display name');

    const registerButton = Array.from(root.querySelectorAll('button'))
      .find((button) => button.textContent?.includes('Create MaxSelf'));
    registerButton?.click();

    const registerRequest = http!.expectOne('http://localhost:8080/api/auth/register');
    registerRequest.flush({ token: 'registered-token' });
    http!.expectOne('http://localhost:8080/api/dashboard').flush(dashboard(25));
    await fixture.whenStable();

    expect(storage['maxself.token']).toBe('registered-token');
    expect(root.textContent).toContain('25 total XP');

    app.logout();
    app.authMode = 'register';
    fixture.detectChanges();
    const retryRegisterButton = Array.from(root.querySelectorAll('button'))
      .find((button) => button.textContent?.includes('Create MaxSelf'));
    retryRegisterButton?.click();
    http!.expectOne('http://localhost:8080/api/auth/register')
      .flush({ error: 'nope' }, { status: 400, statusText: 'Bad Request' });
    await fixture.whenStable();

    expect(root.textContent).toContain('Could not register this account.');
  });

  it('should load an existing session and logout if dashboard loading fails', async () => {
    storage['maxself.token'] = 'stored-token';
    const fixture = TestBed.createComponent(App);
    fixture.detectChanges();

    const dashboardRequest = http!.expectOne('http://localhost:8080/api/dashboard');
    expect(dashboardRequest.request.headers.get('Authorization')).toBe('Bearer stored-token');
    dashboardRequest.flush(dashboard(10));
    await fixture.whenStable();

    const app = fixture.componentInstance;
    expect(app.dashboard()?.progress.totalXp).toBe(10);

    app.loadDashboard();
    http!.expectOne('http://localhost:8080/api/dashboard')
      .flush({ error: 'expired' }, { status: 401, statusText: 'Unauthorized' });
    await fixture.whenStable();

    expect(app.dashboard()).toBeUndefined();
    expect(storage['maxself.token']).toBeUndefined();
  });

  it('should show an activity error and clear saving state when XP claim fails', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    fixture.detectChanges();
    app.token = 'token';
    app.dashboard.set(dashboard());
    app.openActivity(rule);
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const claimButton = Array.from(root.querySelectorAll('button'))
      .find((button) => button.textContent?.includes('Claim XP'));
    claimButton?.click();

    http!.expectOne('http://localhost:8080/api/activities')
      .flush({ error: 'nope' }, { status: 500, statusText: 'Server Error' });
    await fixture.whenStable();

    expect(app.activitySaving()).toBe(false);
    expect(app.activityDialogOpen()).toBe(true);
    expect(root.textContent).toContain('Could not claim XP');
  });

  it('should ignore duplicate auth, duplicate activity save, and missing token dashboard loads', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;

    app.authPending.set(true);
    app.submitAuth();
    app.activitySaving.set(true);
    app.openActivity(rule);
    app.saveActivity();
    app.token = '';
    app.loadDashboard();

    http!.expectNone('http://localhost:8080/api/auth/login');
    http!.expectNone('http://localhost:8080/api/activities');
    http!.expectNone('http://localhost:8080/api/dashboard');
  });
});
