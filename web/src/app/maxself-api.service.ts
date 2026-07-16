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
}

export interface AuthPayload {
  email: string;
  password: string;
  displayName: string;
}

@Injectable({ providedIn: 'root' })
export class MaxSelfApi {
  private readonly http = inject(HttpClient);
  private readonly apiBase = 'http://localhost:8080/api';
  private readonly requestTimeoutMs = 10000;

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

  claimActivity(token: string, type: string, notes: string): Observable<Dashboard> {
    return this.http
      .post<Dashboard>(
        `${this.apiBase}/activities`,
        { type, notes },
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
