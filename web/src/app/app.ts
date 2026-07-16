import { CommonModule } from '@angular/common';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  LucideApple,
  LucideCalendarCheck,
  LucideDroplet,
  LucideDumbbell,
  LucideFlame,
  LucideHeartPulse,
  LucideMoon,
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
import { ActivityRule, AuthMode, Dashboard, MaxSelfApi } from './maxself-api.service';

interface CategoryMeta {
  key: string;
  label: string;
  color: string;
  consistencyKey: string;
}

interface QuestColumn extends CategoryMeta {
  totalXp: number;
  consistencyXp: number;
  rules: ActivityRule[];
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
    LucideHeartPulse,
    LucideMoon,
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
  selectedRule = signal<ActivityRule | undefined>(undefined);
  activityNotes = '';
  activityError = signal('');
  activitySaving = signal(false);

  fallbackRules: ActivityRule[] = [
    { type: 'cardio', title: 'Cardio Session', xp: 30, stat: 'cardio', icon: 'flame', color: '#f59e0b' },
    { type: 'exercise', title: 'Strength Session', xp: 40, stat: 'strength', icon: 'dumbbell', color: '#ff5a5f' },
    { type: 'healthy_meal', title: 'Nourishing Meal', xp: 25, stat: 'fuel', icon: 'apple', color: '#22c55e' },
    { type: 'hydration', title: 'Hydration Boost', xp: 10, stat: 'fuel', icon: 'droplet', color: '#38bdf8' },
    { type: 'sleep', title: 'Sleep Goal Met', xp: 35, stat: 'recovery', icon: 'moon', color: '#6366f1' },
    { type: 'mindfulness', title: 'Mindset Moment', xp: 20, stat: 'mindset', icon: 'sparkles', color: '#a855f7' },
    { type: 'recovery', title: 'Recovery Ritual', xp: 20, stat: 'recovery', icon: 'heart-pulse', color: '#14b8a6' }
  ];

  categoryMeta: CategoryMeta[] = [
    { key: 'cardio', label: 'Cardio', color: '#f59e0b', consistencyKey: 'cardio_consistency' },
    { key: 'strength', label: 'Strength', color: '#ff5a5f', consistencyKey: 'strength_consistency' },
    { key: 'fuel', label: 'Fuel', color: '#22c55e', consistencyKey: 'fuel_consistency' },
    { key: 'recovery', label: 'Recovery', color: '#6366f1', consistencyKey: 'recovery_consistency' },
    { key: 'mindset', label: 'Mindset', color: '#a855f7', consistencyKey: 'mindset_consistency' }
  ];

  statMeta = this.categoryMeta.map(({ key, label, color }) => ({ key, label, color }));

  rules = computed(() => {
    const dashboard = this.dashboard();
    return dashboard?.rules?.length ? dashboard.rules : this.fallbackRules;
  });

  questColumns = computed<QuestColumn[]>(() => {
    const stats = this.dashboard()?.progress.stats ?? {};
    const rules = this.rules();

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

  ngOnInit(): void {
    const url = new URL(window.location.href);
    const callbackToken = url.searchParams.get('token');
    if (callbackToken) {
      this.setToken(callbackToken);
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
    this.selectedRule.set(rule);
    this.activityNotes = '';
    this.activityError.set('');
    this.activityDialogOpen.set(true);
  }

  saveActivity(): void {
    const selectedRule = this.selectedRule();
    if (!selectedRule || this.activitySaving()) {
      return;
    }

    this.activitySaving.set(true);
    this.activityError.set('');

    this.api.claimActivity(this.token, selectedRule.type, this.activityNotes).pipe(
      finalize(() => {
        this.activitySaving.set(false);
      })
    ).subscribe({
      next: (dashboard) => {
        this.dashboard.set(dashboard);
        this.closeActivityDialog();
      },
      error: () => {
        this.activityError.set('Could not claim XP. Please try again in a moment.');
      }
    });
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
    return this.rules().find((rule) => rule.type === type)?.icon ?? 'star';
  }

  colorFor(type: string): string {
    return this.rules().find((rule) => rule.type === type)?.color ?? '#f59e0b';
  }

  closeActivityDialog(): void {
    this.activitySaving.set(false);
    this.activityDialogOpen.set(false);
    this.selectedRule.set(undefined);
    this.activityNotes = '';
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
}
