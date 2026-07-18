import { CommonModule } from '@angular/common';
import { HttpErrorResponse } from '@angular/common/http';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  LucideApple,
  LucideCalendarCheck,
  LucideDroplet,
  LucideDumbbell,
  LucideFlame,
  LucideFootprints,
  LucideHeartPulse,
  LucideMoon,
  LucideRuler,
  LucideScale,
  LucideShield,
  LucideSparkles,
  LucideStar,
  LucideTrophy,
  LucideX,
  LucideZap
} from '@lucide/angular';
import { CardModule } from 'primeng/card';
import { InputTextModule } from 'primeng/inputtext';
import { ProgressBarModule } from 'primeng/progressbar';
import { TableModule } from 'primeng/table';
import { TextareaModule } from 'primeng/textarea';
import { finalize, switchMap, tap } from 'rxjs';
import { ActivityRule, AuthMode, Dashboard, HealthSyncResult, MaxSelfApi, QuestClaim } from './maxself-api.service';

interface CategoryMeta {
  key: string;
  label: string;
  color: string;
  consistencyKey: string;
}

interface QuestColumn extends CategoryMeta {
  totalXp: number;
  consistencyXp: number;
  rules: VisibleActivityRule[];
}

interface VisibleActivityRule extends ActivityRule {
  stackPosition?: number;
  stackTotal?: number;
}

@Component({
  selector: 'app-root',
  imports: [
    CommonModule,
    FormsModule,
    CardModule,
    InputTextModule,
    ProgressBarModule,
    TableModule,
    TextareaModule,
    LucideApple,
    LucideCalendarCheck,
    LucideDroplet,
    LucideDumbbell,
    LucideFlame,
    LucideFootprints,
    LucideHeartPulse,
    LucideMoon,
    LucideRuler,
    LucideScale,
    LucideShield,
    LucideSparkles,
    LucideStar,
    LucideTrophy,
    LucideX,
    LucideZap
  ],
  templateUrl: './app.html',
  styleUrl: './app.scss'
})
export class App implements OnInit {
  private readonly api = inject(MaxSelfApi);

  token = localStorage.getItem('maxself.token') ?? '';
  authMode: AuthMode = 'login';
  email = 'demo@maxself.app';
  password = 'maxself';
  displayName = 'Max Player';

  authError = signal('');
  authPending = signal(false);
  dashboardPending = signal(false);
  dashboard = signal<Dashboard | undefined>(undefined);

  activityDialogOpen = signal(false);
  selectedClaim = signal<QuestClaim | undefined>(undefined);
  claimQueue = signal<QuestClaim[]>([]);
  activityError = signal('');
  activitySaving = signal(false);
  syncError = signal('');
  syncMessage = signal('');
  syncPending = signal(false);
  connectPending = signal(false);
  waistDialogOpen = signal(false);
  waistCentimeters: number | undefined;
  heightCentimeters: number | undefined;
  waistSaving = signal(false);

  fallbackRules: ActivityRule[] = [
    { type: 'cardio', title: 'Cardio Session', xp: 30, stat: 'cardio', icon: 'flame', color: '#f59e0b' },
    { type: 'daily_steps_bronze', title: 'Daily Steps — Bronze', xp: 20, stat: 'cardio', icon: 'footprints', color: '#cd7f32', tier: 'Bronze', thresholdValue: 6000, thresholdUnit: 'steps', followUpType: 'daily_steps_silver' },
    { type: 'daily_steps_silver', title: 'Daily Steps — Silver', xp: 30, stat: 'cardio', icon: 'footprints', color: '#94a3b8', tier: 'Silver', thresholdValue: 8000, thresholdUnit: 'steps', followUpType: 'daily_steps_gold', prerequisiteType: 'daily_steps_bronze' },
    { type: 'daily_steps_gold', title: 'Daily Steps — Gold', xp: 45, stat: 'cardio', icon: 'footprints', color: '#f59e0b', tier: 'Gold', thresholdValue: 10000, thresholdUnit: 'steps', followUpType: 'daily_steps_diamond', prerequisiteType: 'daily_steps_silver' },
    { type: 'daily_steps_diamond', title: 'Daily Steps — Diamond', xp: 70, stat: 'cardio', icon: 'footprints', color: '#67e8f9', tier: 'Diamond', thresholdValue: 15000, thresholdUnit: 'steps', prerequisiteType: 'daily_steps_gold' },
    { type: 'exercise', title: 'Strength Session', xp: 40, stat: 'strength', icon: 'dumbbell', color: '#ff5a5f' },
    { type: 'healthy_meal', title: 'Nourishing Meal', xp: 25, stat: 'fuel', icon: 'apple', color: '#22c55e' },
    { type: 'hydration_bronze', title: 'Hydration Boost — Bronze', xp: 10, stat: 'fuel', icon: 'droplet', color: '#cd7f32', tier: 'Bronze', thresholdValue: 500, thresholdUnit: 'ml', followUpType: 'hydration_silver' },
    { type: 'hydration_silver', title: 'Hydration Boost — Silver', xp: 15, stat: 'fuel', icon: 'droplet', color: '#94a3b8', tier: 'Silver', thresholdValue: 1000, thresholdUnit: 'ml', followUpType: 'hydration_gold', prerequisiteType: 'hydration_bronze' },
    { type: 'hydration_gold', title: 'Hydration Boost — Gold', xp: 20, stat: 'fuel', icon: 'droplet', color: '#f59e0b', tier: 'Gold', thresholdValue: 1500, thresholdUnit: 'ml', followUpType: 'hydration_diamond', prerequisiteType: 'hydration_silver' },
    { type: 'hydration_diamond', title: 'Hydration Boost — Diamond', xp: 30, stat: 'fuel', icon: 'droplet', color: '#67e8f9', tier: 'Diamond', thresholdValue: 2000, thresholdUnit: 'ml', prerequisiteType: 'hydration_gold' },
    { type: 'sleep', title: 'Sleep Goal Met', xp: 35, stat: 'recovery', icon: 'moon', color: '#6366f1' },
    { type: 'mindfulness', title: 'Mindset Moment', xp: 20, stat: 'mindset', icon: 'sparkles', color: '#a855f7' },
    { type: 'recovery', title: 'Recovery Ritual', xp: 20, stat: 'recovery', icon: 'heart-pulse', color: '#14b8a6' },
    { type: 'scale_measurement', title: 'Scale Measurement', xp: 15, stat: 'biometrics', icon: 'scale', color: '#0891b2' },
    { type: 'waist_to_height_ratio', title: 'Waist-to-Height Ratio', xp: 15, stat: 'biometrics', icon: 'ruler', color: '#0891b2' }
  ];

  categoryMeta: CategoryMeta[] = [
    { key: 'cardio', label: 'Cardio', color: '#f59e0b', consistencyKey: 'cardio_consistency' },
    { key: 'strength', label: 'Strength', color: '#ff5a5f', consistencyKey: 'strength_consistency' },
    { key: 'fuel', label: 'Fuel', color: '#22c55e', consistencyKey: 'fuel_consistency' },
    { key: 'recovery', label: 'Recovery', color: '#6366f1', consistencyKey: 'recovery_consistency' },
    { key: 'mindset', label: 'Mindset', color: '#a855f7', consistencyKey: 'mindset_consistency' },
    { key: 'biometrics', label: 'Biometrics', color: '#0891b2', consistencyKey: 'biometrics_consistency' }
  ];

  statMeta = this.categoryMeta.map(({ key, label, color }) => ({ key, label, color }));

  rules = computed(() => {
    const dashboard = this.dashboard();
    return dashboard?.rules?.length ? dashboard.rules : this.fallbackRules;
  });

  questColumns = computed<QuestColumn[]>(() => {
    const stats = this.dashboard()?.progress.stats ?? {};
    const rules = this.visibleQuestRules();

    return this.categoryMeta.map((category) => ({
      ...category,
      totalXp: stats[category.key] || 0,
      consistencyXp: stats[category.consistencyKey] || 0,
      rules: rules.filter((rule) => rule.stat === category.key)
    }));
  });

  progressPercent = computed(() => {
    const progress = this.dashboard()?.progress;
    if (!progress?.nextLevelXp) {
      return 0;
    }
    return Math.round((progress.currentLevelXp / progress.nextLevelXp) * 100);
  });

  todayXp = computed(() => {
    const today = new Date().toDateString();
    return this.dashboard()?.activities
      ?.filter((activity) => new Date(activity.occurredAt).toDateString() === today)
      .reduce((sum, activity) => sum + activity.xp, 0) ?? 0;
  });

  googleHealth = computed(() => this.dashboard()?.googleHealth ?? { connected: false, pendingClaims: 0 });

  pendingClaims = computed(() => this.dashboard()?.questClaims ?? []);

  ngOnInit(): void {
    const url = new URL(window.location.href);
    const callbackToken = url.searchParams.get('token');
    if (callbackToken) {
      this.setToken(callbackToken);
      window.history.replaceState({}, document.title, '/');
    }
    const googleHealth = url.searchParams.get('googleHealth');
    if (googleHealth === 'connected') {
      this.syncMessage.set('Google Health connected. Sync when you are ready to unlock quests.');
      window.history.replaceState({}, document.title, '/');
    } else if (googleHealth === 'error') {
      this.syncError.set('Google Health could not be connected. Please try again.');
      window.history.replaceState({}, document.title, '/');
    }
    if (this.token) {
      this.loadDashboard();
    }
  }

  submitAuth(): void {
    if (this.authPending()) {
      return;
    }

    this.authPending.set(true);
    this.authError.set('');

    this.api.authenticate(this.authMode, {
      email: this.email,
      password: this.password,
      displayName: this.displayName
    }).pipe(
      tap((result) => this.setToken(result.token)),
      switchMap(() => this.api.dashboard(this.token)),
      finalize(() => {
        this.authPending.set(false);
      })
    ).subscribe({
      next: (dashboard) => {
        this.dashboard.set(dashboard);
      },
      error: () => {
        this.clearSession();
        this.authError.set(this.authMode === 'login'
          ? 'Could not log in. Check your credentials or register first.'
          : 'Could not register this account.');
      }
    });
  }

  googleLogin(): void {
    window.location.href = this.api.googleLoginUrl();
  }

  logout(): void {
    this.clearSession();
  }

  openActivity(rule: ActivityRule): void {
    if (rule.type === 'waist_to_height_ratio') {
      this.openWaistDialog();
      return;
    }
    this.syncMessage.set('');
    this.syncError.set('Use Sync Health Data to unlock this quest from Google Health data.');
  }

  saveActivity(): void {
    const claim = this.selectedClaim();
    if (!claim || this.activitySaving()) {
      return;
    }

    this.activitySaving.set(true);
    this.activityError.set('');

    this.api.claimQuest(this.token, claim.id).pipe(
      finalize(() => {
        this.activitySaving.set(false);
      })
    ).subscribe({
      next: (dashboard) => {
        this.dashboard.set(dashboard);
        this.openNextClaim(dashboard);
      },
      error: () => {
        this.activityError.set('Could not claim XP. Please try again in a moment.');
      }
    });
  }

  connectGoogleHealth(): void {
    if (!this.token || this.connectPending()) {
      return;
    }

    this.connectPending.set(true);
    this.syncError.set('');
    this.api.connectGoogleHealth(this.token).pipe(
      finalize(() => {
        this.connectPending.set(false);
      })
    ).subscribe({
      next: (result) => {
        window.location.href = result.url;
      },
      error: (error) => {
        const detail = this.apiErrorMessage(error);
        this.syncError.set(detail ? `Could not connect Google Health: ${detail}` : 'Google Health is not configured yet.');
      }
    });
  }

  syncGoogleHealth(): void {
    if (!this.token || this.syncPending()) {
      return;
    }

    this.syncPending.set(true);
    this.syncError.set('');
    this.syncMessage.set('');

    this.api.syncGoogleHealth(this.token).pipe(
      finalize(() => {
        this.syncPending.set(false);
      })
    ).subscribe({
      next: (result) => {
        this.handleSyncResult(result);
      },
      error: (error) => {
        const detail = this.apiErrorMessage(error);
        if (detail) {
          this.syncError.set(`Could not sync Google Health data: ${detail}`);
          return;
        }
        this.syncError.set(this.googleHealth().connected
          ? 'Could not sync Google Health data. Please try again.'
          : 'Connect Google Health before syncing.');
      }
    });
  }

  openWaistDialog(): void {
    this.waistCentimeters = undefined;
    this.heightCentimeters = undefined;
    this.activityError.set('');
    this.waistDialogOpen.set(true);
  }

  submitWaistMeasurement(): void {
    if (!this.token || this.waistSaving() || !this.waistCentimeters || !this.heightCentimeters) {
      this.activityError.set('Enter both waist and height measurements.');
      return;
    }

    this.waistSaving.set(true);
    this.activityError.set('');
    this.api.submitWaistToHeight(this.token, this.waistCentimeters, this.heightCentimeters).pipe(
      finalize(() => {
        this.waistSaving.set(false);
      })
    ).subscribe({
      next: (result) => {
        this.waistDialogOpen.set(false);
        this.handleSyncResult(result);
      },
      error: () => {
        this.activityError.set('Could not save the measurement. Please try again.');
      }
    });
  }

  openPendingClaims(): void {
    const dashboard = this.dashboard();
    if (!dashboard || !this.pendingClaims().length) {
      return;
    }
    this.handleSyncResult({ createdClaims: 0, pendingClaims: this.pendingClaims(), dashboard });
  }

  loadDashboard(): void {
    if (!this.token || this.dashboardPending()) {
      return;
    }

    this.dashboardPending.set(true);
    this.api.dashboard(this.token).pipe(
      finalize(() => {
        this.dashboardPending.set(false);
      })
    ).subscribe({
      next: (dashboard) => {
        this.dashboard.set(dashboard);
      },
      error: () => {
        this.logout();
      }
    });
  }

  iconFor(type: string): string {
    return this.ruleForType(type)?.icon ?? 'star';
  }

  colorFor(type: string): string {
    return this.ruleForType(type)?.color ?? '#f59e0b';
  }

  questSubtitle(rule: ActivityRule): string {
    if (rule.thresholdValue && rule.thresholdUnit) {
      const stack = this.isStackedQuest(rule) ? ' · stacked quest' : '';
      const followUp = rule.followUpType ? ' · unlocks next tier' : ' · top tier';
      return `${rule.tier ?? 'Tier'} · ${rule.thresholdValue} ${rule.thresholdUnit}${stack}${followUp}`;
    }
    return rule.type === 'waist_to_height_ratio' ? 'Enter measurement' : 'Sync to unlock';
  }

  isStackedQuest(rule: ActivityRule): boolean {
    return Boolean((rule as VisibleActivityRule).stackTotal);
  }

  stackLabel(rule: ActivityRule): string {
    const visible = rule as VisibleActivityRule;
    if (!visible.stackPosition || !visible.stackTotal) {
      return '';
    }
    return `${visible.stackPosition}/${visible.stackTotal}`;
  }

  closeActivityDialog(): void {
    this.activitySaving.set(false);
    this.activityDialogOpen.set(false);
    this.selectedClaim.set(undefined);
    this.activityError.set('');
  }

  closeWaistDialog(): void {
    this.waistSaving.set(false);
    this.waistDialogOpen.set(false);
    this.waistCentimeters = undefined;
    this.heightCentimeters = undefined;
    this.activityError.set('');
  }

  private setToken(token: string): void {
    this.token = token;
    localStorage.setItem('maxself.token', token);
  }

  private clearSession(): void {
    this.token = '';
    this.dashboard.set(undefined);
    localStorage.removeItem('maxself.token');
  }

  private handleSyncResult(result: HealthSyncResult): void {
    this.dashboard.set(result.dashboard);
    const claims = result.pendingClaims?.length ? result.pendingClaims : result.dashboard.questClaims ?? [];
    if (claims.length) {
      this.claimQueue.set(claims);
      this.selectedClaim.set(claims[0]);
      this.activityError.set('');
      this.activityDialogOpen.set(true);
      this.syncMessage.set(result.createdClaims > 0
        ? `${result.createdClaims} new quest${result.createdClaims === 1 ? '' : 's'} unlocked. Claim available tiers in order.`
        : `${claims.length} quest${claims.length === 1 ? '' : 's'} ready to claim.`);
      return;
    }
    this.claimQueue.set([]);
    this.selectedClaim.set(undefined);
    this.activityDialogOpen.set(false);
    this.syncMessage.set('No new quests were unlocked from the latest sync.');
  }

  private openNextClaim(dashboard: Dashboard): void {
    const claimable = dashboard.questClaims ?? [];
    this.claimQueue.set(claimable);
    if (claimable.length) {
      this.selectedClaim.set(claimable[0]);
      this.activityError.set('');
      this.activityDialogOpen.set(true);
      return;
    }
    this.closeActivityDialog();
    this.syncMessage.set('All available quest XP has been claimed.');
  }

  private visibleQuestRules(): VisibleActivityRule[] {
    const rules = this.rules();
    const byType = new Map(rules.map((rule) => [rule.type, rule]));
    const pendingTypes = new Set(this.pendingClaims().map((claim) => claim.type));
    const handled = new Set<string>();
    const visible: VisibleActivityRule[] = [];

    for (const rule of rules) {
      const chain = this.questChain(rule, byType);
      if (!chain.length) {
        visible.push(rule);
        continue;
      }

      const chainKey = chain[0].type;
      if (handled.has(chainKey)) {
        continue;
      }
      chain.forEach((chainRule) => handled.add(chainRule.type));

      const pendingRule = chain.find((chainRule) => pendingTypes.has(chainRule.type));
      const visibleRule = pendingRule ?? chain[0];
      visible.push({
        ...visibleRule,
        stackPosition: chain.findIndex((chainRule) => chainRule.type === visibleRule.type) + 1,
        stackTotal: chain.length
      });
    }

    return visible;
  }

  private questChain(rule: ActivityRule, byType: Map<string, ActivityRule>): ActivityRule[] {
    if (!rule.followUpType && !rule.prerequisiteType) {
      return [];
    }

    let root = rule;
    const seen = new Set<string>();
    while (root.prerequisiteType && byType.has(root.prerequisiteType) && !seen.has(root.type)) {
      seen.add(root.type);
      root = byType.get(root.prerequisiteType)!;
    }

    const chain: ActivityRule[] = [];
    let current: ActivityRule | undefined = root;
    seen.clear();
    while (current && !seen.has(current.type)) {
      chain.push(current);
      seen.add(current.type);
      current = current.followUpType ? byType.get(current.followUpType) : undefined;
    }
    return chain;
  }

  private apiErrorMessage(error: unknown): string | undefined {
    if (!(error instanceof HttpErrorResponse)) {
      return undefined;
    }
    const payload = error.error;
    if (typeof payload === 'string') {
      return payload;
    }
    if (payload && typeof payload.error === 'string') {
      return payload.error;
    }
    return undefined;
  }

  private ruleForType(type: string): ActivityRule | undefined {
    return this.rules().find((rule) => rule.type === type);
  }
}
