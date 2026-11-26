export type DbEngine = 'PostgreSQL' | 'MySQL';
export type DbStatus = 'Healthy' | 'Degraded';
export type DbPermission = 'view_only' | 'view_edit';

export interface TargetDatabase {
  id: string;
  name: string;
  engine: DbEngine;
  driverVersion: string;
  host: string;
  port: number;
  hostPort: string;
  status: DbStatus;
  quarantined: boolean;
  quarantineDraftId?: string;
  headVersion: number;
  redoStack: number[];
}
