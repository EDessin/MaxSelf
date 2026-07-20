import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { provideNoopAnimations } from '@angular/platform-browser/animations';
import { TestBed } from '@angular/core/testing';
import Aura from '@primeuix/themes/aura';
import { providePrimeNG } from 'primeng/config';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { App } from './app';
import { MaxSelfApi } from './maxself-api.service';

const rule = {
  type: 'exercise',
  title: 'Resistance Training',
  xp: 40,
  stat: 'strength',
  icon: 'dumbbell',
  color: '#ff5a5f',
  goal: '10+ min resistance training'
};

const allRules = [
  { type: 'cardio', title: 'Cardio Session', xp: 30, stat: 'cardio', icon: 'flame', color: '#f59e0b', goal: '10+ min cardio' },
  { type: 'daily_steps_bronze', title: 'Daily Steps — Bronze', xp: 20, stat: 'cardio', icon: 'footprints', color: '#f59e0b', goal: '6000 steps', tier: 'Bronze', thresholdValue: 6000, thresholdUnit: 'steps', followUpType: 'daily_steps_silver' },
  { type: 'daily_steps_silver', title: 'Daily Steps — Silver', xp: 30, stat: 'cardio', icon: 'footprints', color: '#f59e0b', goal: '8000 steps', tier: 'Silver', thresholdValue: 8000, thresholdUnit: 'steps', followUpType: 'daily_steps_gold', prerequisiteType: 'daily_steps_bronze' },
  { type: 'daily_steps_gold', title: 'Daily Steps — Gold', xp: 45, stat: 'cardio', icon: 'footprints', color: '#f59e0b', goal: '10000 steps', tier: 'Gold', thresholdValue: 10000, thresholdUnit: 'steps', followUpType: 'daily_steps_diamond', prerequisiteType: 'daily_steps_silver' },
  { type: 'daily_steps_diamond', title: 'Daily Steps — Diamond', xp: 70, stat: 'cardio', icon: 'footprints', color: '#f59e0b', goal: '15000 steps', tier: 'Diamond', thresholdValue: 15000, thresholdUnit: 'steps', prerequisiteType: 'daily_steps_gold' },
  rule,
  { type: 'mobility', title: 'Mobility Session', xp: 20, stat: 'strength', icon: 'person-standing', color: '#ff5a5f', goal: '10+ min mobility' },
  { type: 'healthy_meal', title: 'Nourishing Meal', xp: 25, stat: 'fuel', icon: 'apple', color: '#22c55e', goal: 'Log nutrition' },
  { type: 'hydration_bronze', title: 'Hydration Boost — Bronze', xp: 10, stat: 'fuel', icon: 'droplet', color: '#22c55e', goal: '500 ml', tier: 'Bronze', thresholdValue: 500, thresholdUnit: 'ml', followUpType: 'hydration_silver' },
  { type: 'hydration_silver', title: 'Hydration Boost — Silver', xp: 15, stat: 'fuel', icon: 'droplet', color: '#22c55e', goal: '1000 ml', tier: 'Silver', thresholdValue: 1000, thresholdUnit: 'ml', followUpType: 'hydration_gold', prerequisiteType: 'hydration_bronze' },
  { type: 'hydration_gold', title: 'Hydration Boost — Gold', xp: 20, stat: 'fuel', icon: 'droplet', color: '#22c55e', goal: '1500 ml', tier: 'Gold', thresholdValue: 1500, thresholdUnit: 'ml', followUpType: 'hydration_diamond', prerequisiteType: 'hydration_silver' },
  { type: 'hydration_diamond', title: 'Hydration Boost — Diamond', xp: 30, stat: 'fuel', icon: 'droplet', color: '#22c55e', goal: '2000 ml', tier: 'Diamond', thresholdValue: 2000, thresholdUnit: 'ml', prerequisiteType: 'hydration_gold' },
  { type: 'sleep', title: 'Sleep Goal Met', xp: 35, stat: 'recovery', icon: 'moon', color: '#6366f1', goal: '7 hours', thresholdValue: 7, thresholdUnit: 'hours' },
  { type: 'mindfulness', title: 'Mindset Moment', xp: 20, stat: 'mindset', icon: 'sparkles', color: '#a855f7', goal: 'not ready yet' },
  { type: 'recovery', title: 'Recovery Ritual', xp: 20, stat: 'recovery', icon: 'stretch-horizontal', color: '#6366f1', goal: '10+ min stretching' },
  { type: 'scale_measurement', title: 'Scale Measurement', xp: 15, stat: 'biometrics', icon: 'scale', color: '#0891b2', goal: 'Weight or body fat' },
  { type: 'waist_to_height_ratio', title: 'Waist-to-Height Ratio', xp: 15, stat: 'biometrics', icon: 'ruler', color: '#0891b2', goal: 'Waist + height' },
  { type: 'bonus', title: 'Bonus Quest', xp: 5, stat: 'mindset', icon: 'unknown', color: '#f59e0b' }
];

const visibleRuleCount = allRules.length - 6;

const claim = {
  id: 'claim-1',
  userId: 'user-1',
  type: 'cardio',
  title: 'Cardio Session',
  xp: 30,
  stat: 'cardio',
  source: 'google_health',
  sourceId: 'run-1',
  evidence: 'Running · 35 min',
  occurredAt: new Date().toISOString(),
  questDate: '2026-07-17',
  status: 'pending',
  createdAt: new Date().toISOString()
};

const secondClaim = {
  ...claim,
  id: 'claim-2',
  type: 'sleep',
  title: 'Sleep Goal Met',
  xp: 35,
  stat: 'recovery',
  sourceId: 'sleep-1',
  evidence: '7 hours 30 minutes asleep'
};

function questDateKey(date = new Date()) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

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
        biometrics: 0,
        cardio_consistency: 0,
        strength_consistency: 0,
        fuel_consistency: 0,
        recovery_consistency: 0,
        mindset_consistency: 0,
        biometrics_consistency: 0
      }
    },
    activities: [],
    rules: [rule],
    googleHealth: {
      connected: false,
      pendingClaims: 0
    },
    questClaims: [],
    questClaimHistory: []
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
        biometrics: 30,
        cardio_consistency: 5,
        strength_consistency: 10,
        fuel_consistency: 15,
        recovery_consistency: 20,
        mindset_consistency: 25,
        biometrics_consistency: 30,
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
    expect(root.textContent).toContain('Strength & mobility');
    expect(root.textContent).toContain('Fuel');
    expect(root.textContent).toContain('Biometrics');
    expect(root.textContent).not.toContain('99 XP');
    expect(root.textContent).toContain('Recent Wins');
    expect(root.textContent).toContain('Google Health Sync');
    expect(root.textContent).toContain('Connect Google Health');
    expect(Array.from(root.querySelectorAll('.pillar-category-name')).map((header) => header.textContent?.trim()))
      .toEqual(['Cardio', 'Strength & mobility', 'Fuel', 'Recovery', 'Mindset', 'Biometrics']);
    expect(Array.from(root.querySelectorAll('.pillar-heading')).map((header) => [
      header.querySelector('.pillar-category-name')?.textContent?.trim(),
      header.querySelector('.pillar-xp-summary strong')?.textContent?.trim(),
      header.querySelector('.pillar-xp-summary small')?.textContent?.trim()
    ].join(' ')))
      .toEqual([
        'Cardio 30 total XP 5 consistency XP',
        'Strength & mobility 40 total XP 10 consistency XP',
        'Fuel 35 total XP 15 consistency XP',
        'Recovery 55 total XP 20 consistency XP',
        'Mindset 25 total XP 25 consistency XP',
        'Biometrics 30 total XP 30 consistency XP'
      ]);
    expect(root.querySelectorAll('.quest-column-header').length).toBe(0);
    expect(root.textContent).toContain('Scale Measurement');
    expect(root.textContent).toContain('Waist-to-Height Ratio');
    expect(root.textContent).toContain('10+ min cardio');
    expect(root.textContent).toContain('Resistance Training');
    expect(root.textContent).toContain('10+ min resistance training');
    expect(root.textContent).toContain('Mobility Session');
    expect(root.textContent).toContain('10+ min mobility');
    expect(root.textContent).toContain('Log nutrition');
    expect(root.textContent).toContain('7 hours');
    expect(root.textContent).toContain('6000 steps');
    expect(root.textContent).toContain('500 ml');
    expect(root.textContent).toContain('not ready yet');
    expect(root.textContent).toContain('10+ min stretching');
    expect(root.textContent).toContain('Weight or body fat');
    expect(root.textContent).toContain('Waist + height');
    expect(root.textContent).not.toContain('Bronze · 6000 steps');
    expect(root.textContent).not.toContain('Bronze · 500 ml');
    expect(root.textContent).not.toContain(`Sync to ${'unlock'}`);
    expect(root.textContent).not.toContain('Lab Results');
    expect(root.textContent).not.toContain('Body Composition Scan');
    expect(root.querySelectorAll('.action-tile').length).toBe(visibleRuleCount);
    expect(root.querySelectorAll('.quest-stack').length).toBe(2);
    expect(root.querySelectorAll('.stack-pips').length).toBe(2);
    expect(root.querySelector('.hero-metrics')).not.toBeNull();
    expect(root.querySelector('.hero-performance-art')?.getAttribute('src')).toBe('/quest-art/hero-performance.svg');
    expect(Array.from(root.querySelectorAll<HTMLImageElement>('.pillar-art')).map((image) => image.getAttribute('src')))
      .toEqual([
        '/quest-art/pillar-cardio.svg',
        '/quest-art/pillar-strength.svg',
        '/quest-art/pillar-fuel.svg',
        '/quest-art/pillar-recovery.svg',
        '/quest-art/pillar-mindset.svg',
        '/quest-art/pillar-biometrics.svg'
      ]);
    expect(Array.from(root.querySelectorAll('.action-tile')).map((tile) => tile.textContent).join('\n'))
      .not.toContain('Daily Steps — Silver');
    expect(root.querySelectorAll('.quest-column').length).toBe(6);
    expect(root.querySelectorAll('tbody tr').length).toBe(allRules.length);
    expect(app.progressPercent()).toBe(20);
    expect(app.todayXp()).toBe(allRules.reduce((sum, activityRule) => sum + activityRule.xp, 0));
    expect(app.iconFor('missing')).toBe('star');
    expect(app.colorFor('missing')).toBe('#f59e0b');
    expect(app.iconFor('daily_steps_bronze')).toBe('footprints');
    expect(app.colorFor('hydration_bronze')).toBe('#22c55e');
    expect(app.colorFor('recovery')).toBe('#6366f1');
    expect(app.rules().find((dashboardRule) => dashboardRule.type === 'bonus')?.color).toBe('#a855f7');
  });

  it('should open and close the waist measurement dialog from the waist quest', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.dashboard.set(fullDashboard());
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const action = Array.from(root.querySelectorAll<HTMLButtonElement>('.action-tile'))
      .find((button) => button.textContent?.includes('Waist-to-Height Ratio'));
    action?.click();
    await fixture.whenStable();

    expect(root.querySelector('.activity-dialog')).not.toBeNull();
    expect(root.textContent).toContain('Waist (cm)');

    const closeButton = root.querySelector('button[aria-label="Close waist measurement dialog"]') as HTMLButtonElement;
    closeButton.click();
    await fixture.whenStable();

    expect(root.querySelector('.activity-dialog')).toBeNull();
    expect(app.waistDialogOpen()).toBe(false);
  });

  it('should show only the currently claimable tier as a stacked tile', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.dashboard.set({
      ...fullDashboard(),
      questClaims: [{
        ...claim,
        type: 'daily_steps_silver',
        title: 'Daily Steps — Silver',
        xp: 30,
        sourceId: 'steps-silver',
        evidence: '8400 steps'
      }]
    });
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const dailyStepsElement = Array.from(root.querySelectorAll<HTMLElement>('.action-tile'))
      .find((tile) => tile.textContent?.includes('Daily Steps'));
    const tileText = Array.from(root.querySelectorAll<HTMLElement>('.action-tile'))
      .map((tile) => tile.textContent?.replace(/\s+/g, ' ').trim() ?? '');
    const dailyStepsTile = tileText.find((text) => text.includes('Daily Steps'));

    expect(dailyStepsTile).toContain('Daily Steps — Silver');
    expect(dailyStepsTile).toContain('8000 steps');
    expect(dailyStepsTile).not.toContain('Silver · 8000 steps');
    expect(dailyStepsTile).not.toContain('2/4');
    expect(dailyStepsElement?.querySelectorAll('.stack-pip.filled').length).toBe(2);
    expect(dailyStepsElement?.querySelector('.tier-marker.silver')).not.toBeNull();
    expect(dailyStepsTile).not.toContain('Daily Steps — Bronze');
    expect(tileText.filter((text) => text.includes('Daily Steps')).length).toBe(1);
  });

  it('should advance the stacked tile after a lower tier was claimed today', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.dashboard.set({
      ...fullDashboard(),
      questClaims: [],
      questClaimHistory: [{
        ...claim,
        type: 'daily_steps_bronze',
        title: 'Daily Steps — Bronze',
        xp: 20,
        sourceId: 'steps-bronze',
        evidence: '6500 steps',
        questDate: questDateKey(),
        status: 'claimed',
        claimedAt: new Date().toISOString()
      }]
    });
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const dailyStepsElement = Array.from(root.querySelectorAll<HTMLElement>('.action-tile'))
      .find((tile) => tile.textContent?.includes('Daily Steps'));
    const dailyStepsTile = dailyStepsElement?.textContent?.replace(/\s+/g, ' ').trim() ?? '';

    expect(dailyStepsTile).toContain('Daily Steps — Silver');
    expect(dailyStepsTile).toContain('8000 steps');
    expect(dailyStepsTile).not.toContain('Silver · 8000 steps');
    expect(dailyStepsTile).not.toContain('Daily Steps — Bronze');
    expect(dailyStepsElement?.querySelectorAll('.stack-pip.filled').length).toBe(2);

    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    app.dashboard.set({
      ...fullDashboard(),
      questClaims: [],
      questClaimHistory: [{
        ...claim,
        type: 'daily_steps_bronze',
        title: 'Daily Steps — Bronze',
        xp: 20,
        sourceId: 'steps-bronze-yesterday',
        evidence: '6500 steps',
        questDate: questDateKey(yesterday),
        status: 'claimed',
        claimedAt: yesterday.toISOString()
      }]
    });
    fixture.detectChanges();

    const resetDailyStepsTile = Array.from(root.querySelectorAll<HTMLElement>('.action-tile'))
      .find((tile) => tile.textContent?.includes('Daily Steps'))
      ?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
    expect(resetDailyStepsTile).toContain('Daily Steps — Bronze');
    expect(resetDailyStepsTile).not.toContain('Daily Steps — Silver');
  });

  it('should close the claim dialog after a successful XP claim', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.token = 'token';
    app.dashboard.set({ ...dashboard(), questClaims: [claim] });
    app.claimQueue.set([claim]);
    app.selectedClaim.set(claim);
    app.activityDialogOpen.set(true);

    app.saveActivity();

    const request = http!.expectOne('http://localhost:8080/api/quest-claims/claim-1/claim');
    expect(request.request.method).toBe('POST');
    expect(app.activitySaving()).toBe(true);

    request.flush(dashboard(30));

    expect(app.activitySaving()).toBe(false);
    expect(app.activityDialogOpen()).toBe(false);
    expect(app.selectedClaim()).toBeUndefined();
    expect(app.dashboard()?.progress.totalXp).toBe(30);
  });

  it('should remove the rendered activity modal after clicking Claim XP successfully', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.dashboard.set({ ...dashboard(), questClaims: [claim] });
    app.claimQueue.set([claim]);
    app.selectedClaim.set(claim);
    app.activityDialogOpen.set(true);
    fixture.detectChanges();
    app.token = 'token';

    const root = fixture.nativeElement as HTMLElement;
    expect(root.querySelector('.activity-dialog')).not.toBeNull();

    const claimButton = Array.from(root.querySelectorAll('button'))
      .find((button) => button.textContent?.includes('Claim XP'));
    claimButton?.click();

    const request = http!.expectOne('http://localhost:8080/api/quest-claims/claim-1/claim');
    request.flush(dashboard(30));
    await fixture.whenStable();

    expect(root.querySelector('.modal-backdrop')).toBeNull();
    expect(root.querySelector('.activity-dialog')).toBeNull();
  });

  it('should make old pending claim dates clear in the XP dialog', () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    const root = fixture.nativeElement as HTMLElement;

    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const yesterdayClaim = {
      ...claim,
      questDate: questDateKey(yesterday),
      occurredAt: yesterday.toISOString()
    };

    app.dashboard.set({ ...dashboard(), questClaims: [yesterdayClaim] });
    app.claimQueue.set([yesterdayClaim]);
    app.selectedClaim.set(yesterdayClaim);
    app.activityDialogOpen.set(true);
    fixture.detectChanges();

    expect(root.textContent).toContain(`Yesterday's quest · ${app.claimDateLabel(yesterdayClaim)}`);
    expect(root.textContent).toContain('This XP comes from synced data for a previous day.');
    expect(root.textContent).toContain(`Claim XP for ${app.claimDateLabel(yesterdayClaim)}`);

    const older = new Date();
    older.setDate(older.getDate() - 3);
    const olderClaim = {
      ...claim,
      questDate: questDateKey(older),
      occurredAt: older.toISOString()
    };
    app.selectedClaim.set(olderClaim);
    fixture.detectChanges();

    expect(root.textContent).toContain(`Past quest · ${app.claimDateLabel(olderClaim)}`);
    expect(root.textContent).not.toContain("Yesterday's quest");

    const todayClaim = {
      ...claim,
      questDate: questDateKey(),
      occurredAt: new Date().toISOString()
    };
    app.selectedClaim.set(todayClaim);
    fixture.detectChanges();

    expect(root.textContent).not.toContain('This XP comes from synced data for a previous day.');
    expect(root.textContent).not.toContain('Past quest');
    expect(root.textContent).not.toContain("Yesterday's quest");
    expect(root.textContent).toContain('Claim XP');
    expect(root.textContent).not.toContain('Claim XP for');
  });

  it('should sync Google Health and open the first claimable quest', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    app.dashboard.set({
      ...dashboard(),
      googleHealth: { connected: true, pendingClaims: 0 }
    });
    fixture.detectChanges();
    app.token = 'token';

    const root = fixture.nativeElement as HTMLElement;
    const syncButton = Array.from(root.querySelectorAll('button'))
      .find((button) => button.textContent?.includes('Sync Health Data'));
    syncButton?.click();

    const request = http!.expectOne('http://localhost:8080/api/integrations/google-health/sync');
    expect(request.request.method).toBe('POST');
    request.flush({
      createdClaims: 1,
      pendingClaims: [claim],
      dashboard: {
        ...dashboard(),
        googleHealth: { connected: true, pendingClaims: 1 },
        questClaims: [claim]
      }
    });
    await fixture.whenStable();

    expect(root.querySelector('.activity-dialog')).not.toBeNull();
    expect(root.textContent).toContain('Running · 35 min');
    expect(root.textContent).toContain('1 new quest unlocked. Claim available tiers in order.');
  });

  it('should connect Google Health and surface configuration errors', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    fixture.detectChanges();
    app.token = 'token';

    app.connectGoogleHealth();

    let request = http!.expectOne('http://localhost:8080/api/integrations/google-health/connect');
    expect(request.request.method).toBe('POST');
    request.flush({ url: `${window.location.origin}${window.location.pathname}` });

    expect(app.connectPending()).toBe(false);

    app.connectGoogleHealth();
    request = http!.expectOne('http://localhost:8080/api/integrations/google-health/connect');
    request.flush({ error: 'missing config' }, { status: 501, statusText: 'Not Implemented' });

    expect(app.connectPending()).toBe(false);
    expect(app.syncError()).toBe('Could not connect Google Health: missing config');
  });

  it('should report sync failures and empty sync results', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    fixture.detectChanges();
    app.token = 'token';
    app.dashboard.set({
      ...dashboard(),
      googleHealth: { connected: false, pendingClaims: 0 }
    });

    app.syncGoogleHealth();

    let request = http!.expectOne('http://localhost:8080/api/integrations/google-health/sync');
    request.flush({ error: 'not connected' }, { status: 409, statusText: 'Conflict' });
    await fixture.whenStable();

    expect(app.syncPending()).toBe(false);
    expect(app.syncError()).toBe('Could not sync Google Health data: not connected');

    app.dashboard.set({
      ...dashboard(),
      googleHealth: { connected: true, pendingClaims: 0 }
    });
    app.syncGoogleHealth();

    request = http!.expectOne('http://localhost:8080/api/integrations/google-health/sync');
    request.flush({
      createdClaims: 0,
      pendingClaims: [],
      dashboard: {
        ...dashboard(),
        googleHealth: { connected: true, pendingClaims: 0 },
        questClaims: []
      }
    });
    await fixture.whenStable();

    expect(app.activityDialogOpen()).toBe(false);
    expect(app.selectedClaim()).toBeUndefined();
    expect(app.syncMessage()).toBe('No new quests were unlocked from the latest sync.');
  });

  it('should submit a waist measurement and open the generated claim', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    fixture.detectChanges();
    app.token = 'token';
    app.dashboard.set(dashboard());

    app.submitWaistMeasurement();
    expect(app.activityError()).toBe('Enter both waist and height measurements.');
    http!.expectNone('http://localhost:8080/api/biometrics/waist-to-height');

    app.waistCentimeters = 80;
    app.heightCentimeters = 180;
    app.submitWaistMeasurement();

    const request = http!.expectOne('http://localhost:8080/api/biometrics/waist-to-height');
    expect(request.request.method).toBe('POST');
    expect(request.request.body).toEqual({ waistCentimeters: 80, heightCentimeters: 180 });
    request.flush({
      createdClaims: 1,
      pendingClaims: [{
        ...claim,
        id: 'waist-claim',
        type: 'waist_to_height_ratio',
        title: 'Waist-to-Height Ratio',
        xp: 15,
        stat: 'biometrics',
        source: 'manual',
        evidence: 'Waist 80.0 cm, height 180.0 cm, ratio 0.44'
      }],
      dashboard: {
        ...dashboard(),
        questClaims: []
      }
    });
    await fixture.whenStable();

    expect(app.waistSaving()).toBe(false);
    expect(app.waistDialogOpen()).toBe(false);
    expect(app.activityDialogOpen()).toBe(true);
    expect(app.selectedClaim()?.type).toBe('waist_to_height_ratio');
  });

  it('should show a waist measurement error when saving fails', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    fixture.detectChanges();
    app.token = 'token';
    app.waistCentimeters = 80;
    app.heightCentimeters = 180;

    app.submitWaistMeasurement();

    http!.expectOne('http://localhost:8080/api/biometrics/waist-to-height')
      .flush({ error: 'nope' }, { status: 500, statusText: 'Server Error' });
    await fixture.whenStable();

    expect(app.waistSaving()).toBe(false);
    expect(app.activityError()).toBe('Could not save the measurement. Please try again.');
  });

  it('should open existing pending claims and advance through a multi-claim queue', async () => {
    const fixture = TestBed.createComponent(App);
    const app = fixture.componentInstance;
    fixture.detectChanges();
    app.token = 'token';

    app.openPendingClaims();
    expect(app.activityDialogOpen()).toBe(false);

    app.dashboard.set({ ...dashboard(), questClaims: [claim, secondClaim] });
    app.openPendingClaims();

    expect(app.activityDialogOpen()).toBe(true);
    expect(app.selectedClaim()?.id).toBe('claim-1');

    app.saveActivity();
    http!.expectOne('http://localhost:8080/api/quest-claims/claim-1/claim')
      .flush({ ...dashboard(30), questClaims: [secondClaim] });
    await fixture.whenStable();

    expect(app.activityDialogOpen()).toBe(true);
    expect(app.selectedClaim()?.id).toBe('claim-2');
  });

  it('should expose the Google login URL from the API service', () => {
    const api = TestBed.inject(MaxSelfApi);
    expect(api.googleLoginUrl()).toBe('http://localhost:8080/api/auth/google/login');
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
    app.dashboard.set({ ...dashboard(), questClaims: [claim] });
    app.claimQueue.set([claim]);
    app.selectedClaim.set(claim);
    app.activityDialogOpen.set(true);
    fixture.detectChanges();

    const root = fixture.nativeElement as HTMLElement;
    const claimButton = Array.from(root.querySelectorAll('button'))
      .find((button) => button.textContent?.includes('Claim XP'));
    claimButton?.click();
    fixture.detectChanges();

    expect(root.querySelector('.claim-impact')).not.toBeNull();
    expect(root.querySelector('.activity-dialog')?.classList.contains('claim-ritual-active')).toBe(true);

    http!.expectOne('http://localhost:8080/api/quest-claims/claim-1/claim')
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
    app.selectedClaim.set(claim);
    app.activityDialogOpen.set(true);
    app.saveActivity();
    app.token = '';
    app.loadDashboard();

    http!.expectNone('http://localhost:8080/api/auth/login');
    http!.expectNone('http://localhost:8080/api/quest-claims/claim-1/claim');
    http!.expectNone('http://localhost:8080/api/dashboard');
  });
});
