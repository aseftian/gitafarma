export interface Medicine {
  id: string;
  name: string;
  barcode: string;
  sku: string;
  price: number;
  uom: string;
  alt_uom: string;
  updated_at: string;
}

export interface MedicinePage {
  items: Medicine[];
  total: number;
}

export interface RecentMedicinesResponse extends MedicinePage {
  recent_days: number;
}

export interface Setting {
  key: string;
  value: string;
  description: string;
  updated_at: string;
}

export interface AdminUser {
  email: string;
  name: string;
  role: string;
  active?: boolean;
}

export interface AuthSessionResponse {
  admin: AdminUser;
  expires_at?: string;
  csrf_token: string;
}

export interface SyncResult {
  total_fetched: number;
  upserted: number;
  deleted: number;
  price_changes: number;
}

export interface AuditLog {
  id: number;
  admin_email: string;
  action: string;
  entity_type: string;
  entity_id: string;
  detail: string;
  created_at: string;
}
