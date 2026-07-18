import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, timeout } from 'rxjs';

export type AuthMode = 'login' | 'register';

export interface User {
  id: string;
  email: string;
  displayName: string;
  avatarUrl?: string;
}

export interface ActivityRule {
  type: string;
  title: string;
  xp: number;
  stat: string;
  icon: string;
  color: string;
  tier?: string;
  thresholdValue?: number;
  thresholdUnit?: string;
  followUpType?: string;
  prerequisiteType?: string;
}

export interface Activity {
  id: string;
  type: string;
  title: string;
  notes: string;
  xp: number;
  stat: string;
  occurredAt: string;
}

export interface GoogleHealthStatus {
  connected: boolean;
  lastSyncedAt?: string;
  pendingClaims: number;
}

export interface QuestClaim {
  id: string;
  type: string;
  title: string;
  xp: number;
  stat: string;
  source: string;
  sourceId: string;
  evidence: string;
  occurredAt: string;
  questDate: string;
  status: string;
  activityId?: string;
  createdAt: string;
  claimedAt?: string;
}

export interface ProgressProfile {
  level: number;
  totalXp: number;
  currentLevelXp: number;
  nextLevelXp: number;
  streakDays: number;
  stats: Record<string, number>;
}

export interface Dashboard {
  user: User;
  progress: ProgressProfile;
  activities: Activity[];
  rules: ActivityRule[];
  googleHealth?: GoogleHealthStatus;
  questClaims?: QuestClaim[];
}

export interface AuthPayload {
  email: string;
  password: string;
  displayName: string;
}

export interface HealthSyncResult {
  createdClaims: number;
  pendingClaims: QuestClaim[];
  dashboard: Dashboard;
}

@Injectable({ providedIn: 'root' })
export class MaxSelfApi {
  private readonly http = inject(HttpClient);
  private readonly apiBase = 'http://localhost:8080/api';
  private readonly requestTimeoutMs = 10000;
  private readonly healthSyncTimeoutMs = 180000;

  authenticate(mode: AuthMode, payload: AuthPayload): Observable<{ token: string }> {
    const path = mode === 'login' ? '/auth/login' : '/auth/register';
    return this.http
      .post<{ token: string }>(`${this.apiBase}${path}`, payload)
      .pipe(timeout(this.requestTimeoutMs));
  }

  dashboard(token: string): Observable<Dashboard> {
    return this.http
      .get<Dashboard>(`${this.apiBase}/dashboard`, { headers: this.authHeaders(token) })
      .pipe(timeout(this.requestTimeoutMs));
  }

  connectGoogleHealth(token: string): Observable<{ url: string }> {
    return this.http
      .post<{ url: string }>(
        `${this.apiBase}/integrations/google-health/connect`,
        {},
        { headers: this.authHeaders(token) }
      )
      .pipe(timeout(this.requestTimeoutMs));
  }

  syncGoogleHealth(token: string): Observable<HealthSyncResult> {
    return this.http
      .post<HealthSyncResult>(
        `${this.apiBase}/integrations/google-health/sync`,
        {},
        { headers: this.authHeaders(token) }
      )
      .pipe(timeout(this.healthSyncTimeoutMs));
  }

  submitWaistToHeight(token: string, waistCentimeters: number, heightCentimeters: number): Observable<HealthSyncResult> {
    return this.http
      .post<HealthSyncResult>(
        `${this.apiBase}/biometrics/waist-to-height`,
        { waistCentimeters, heightCentimeters },
        { headers: this.authHeaders(token) }
      )
      .pipe(timeout(this.requestTimeoutMs));
  }

  claimQuest(token: string, claimId: string): Observable<Dashboard> {
    return this.http
      .post<Dashboard>(
        `${this.apiBase}/quest-claims/${claimId}/claim`,
        {},
        { headers: this.authHeaders(token) }
      )
      .pipe(timeout(this.requestTimeoutMs));
  }

  googleLoginUrl(): string {
    return `${this.apiBase}/auth/google/login`;
  }

  private authHeaders(token: string): HttpHeaders {
    return new HttpHeaders({ Authorization: `Bearer ${token}` });
  }
}
