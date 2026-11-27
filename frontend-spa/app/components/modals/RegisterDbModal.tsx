import React, { useState } from 'react';
import type { DbEngine, TargetDatabase } from '../../types';
import { M3Dialog } from '../common/M3Dialog';

interface SaveDatabasePayload {
  name: string;
  engine: DbEngine;
  host: string;
  port: string;
}

interface RegisterDbModalProps {
  open: boolean;
  onClose: () => void;
  editingDb?: TargetDatabase | null;
  onSave: (dbData: SaveDatabasePayload) => void;
}

export const RegisterDbModal: React.FC<RegisterDbModalProps> = ({
  open,
  onClose,
  editingDb,
  onSave,
}) => {
  const dialogTitle = editingDb ? `Edit ${editingDb.name}` : 'Register Target Database';
  const submitLabel = editingDb ? 'Save Changes' : 'Register DB';

  return (
    <M3Dialog
      open={open}
      onClose={onClose}
      title={dialogTitle}
      maxWidth="max-w-md"
      footer={
        <ModalActionButtons onClose={onClose} submitFormId="register-db-form" submitLabel={submitLabel} />
      }
    >
      <RegisterDbForm
        key={editingDb?.id ?? 'new-db'}
        editingDb={editingDb}
        onSave={onSave}
      />
    </M3Dialog>
  );
};

// --- Form Component (Stepdown Rule) ---

interface RegisterDbFormProps {
  editingDb?: TargetDatabase | null;
  onSave: (dbData: SaveDatabasePayload) => void;
}

const RegisterDbForm: React.FC<RegisterDbFormProps> = ({ editingDb, onSave }) => {
  const [fields, setFields] = useState<SaveDatabasePayload>(() => buildInitialDbFields(editingDb));
  const [errorMessage, setErrorMessage] = useState('');

  const handleFieldChange = (key: keyof SaveDatabasePayload, value: string) => {
    setFields((prev) => ({ ...prev, [key]: value }));
    if (errorMessage) setErrorMessage('');
  };

  const handleEngineChange = (engine: DbEngine) => {
    const defaultPort = engine === 'PostgreSQL' ? '5432' : '3306';
    setFields((prev) => ({ ...prev, engine, port: defaultPort }));
  };

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    const validationError = validateDbFields(fields);

    if (validationError) {
      setErrorMessage(validationError);
      return;
    }

    setErrorMessage('');
    onSave({
      name: fields.name.trim(),
      engine: fields.engine,
      host: fields.host.trim(),
      port: fields.port.trim(),
    });
  };

  return (
    <form id="register-db-form" onSubmit={handleSubmit} className="space-y-3.5">
      <div>
        <label htmlFor="db-name" className="block text-xs font-bold text-[#3A241A] mb-1">
          Database Name
        </label>
        <input
          id="db-name"
          type="text"
          value={fields.name}
          onChange={(e) => handleFieldChange('name', e.target.value)}
          placeholder="inventory_prod_pg"
          className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] font-mono text-xs px-3.5 focus:outline-none focus:border-[#1E6B8C] focus:ring-2 focus:ring-[#1E6B8C]/20"
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor="db-engine" className="block text-xs font-bold text-[#3A241A] mb-1">
            Engine
          </label>
          <select
            id="db-engine"
            value={fields.engine}
            onChange={(e) => handleEngineChange(e.target.value as DbEngine)}
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] text-xs font-semibold px-3 focus:outline-none focus:border-[#1E6B8C] cursor-pointer"
          >
            <option value="PostgreSQL">PostgreSQL</option>
            <option value="MySQL">MySQL</option>
          </select>
        </div>

        <div>
          <label htmlFor="db-port" className="block text-xs font-bold text-[#3A241A] mb-1">
            Port
          </label>
          <input
            id="db-port"
            type="number"
            value={fields.port}
            onChange={(e) => handleFieldChange('port', e.target.value)}
            className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] font-mono text-xs px-3.5 focus:outline-none focus:border-[#1E6B8C]"
          />
        </div>
      </div>

      <div>
        <label htmlFor="db-host" className="block text-xs font-bold text-[#3A241A] mb-1">
          Host
        </label>
        <input
          id="db-host"
          type="text"
          value={fields.host}
          onChange={(e) => handleFieldChange('host', e.target.value)}
          placeholder="pg-inventory-01.internal"
          className="w-full h-10 rounded-xl border border-[#D8C7A5] bg-[#FFFFFF] text-[#3A241A] placeholder-[#8A7361] font-mono text-xs px-3.5 focus:outline-none focus:border-[#1E6B8C]"
        />
      </div>

      {errorMessage && <p className="text-xs font-semibold text-[#822433]">{errorMessage}</p>}
    </form>
  );
};

// --- Low-Level Technical Helpers (Stepdown Rule) ---

function buildInitialDbFields(editingDb?: TargetDatabase | null): SaveDatabasePayload {
  if (!editingDb) {
    return { name: '', engine: 'PostgreSQL', host: '', port: '5432' };
  }
  const [host = '', port = ''] = editingDb.hostPort.split(':');
  const defaultPort = editingDb.engine === 'PostgreSQL' ? '5432' : '3306';

  return {
    name: editingDb.name,
    engine: editingDb.engine,
    host,
    port: port || defaultPort,
  };
}

function validateDbFields(fields: SaveDatabasePayload): string | null {
  if (!fields.name.trim()) return 'Database name is required.';
  if (!fields.host.trim()) return 'Host is required.';
  if (!fields.port.trim()) return 'Port is required.';
  return null;
}

interface ModalActionButtonsProps {
  onClose: () => void;
  submitFormId: string;
  submitLabel: string;
}

const ModalActionButtons: React.FC<ModalActionButtonsProps> = ({
  onClose,
  submitFormId,
  submitLabel,
}) => (
  <>
    <button
      type="button"
      onClick={onClose}
      className="h-10 px-4 rounded-xl border border-[#D8C7A5] bg-[#FFFDF9] hover:bg-[#EFE5D1] text-[#3A241A] text-xs font-semibold cursor-pointer"
    >
      Cancel
    </button>
    <button
      type="submit"
      form={submitFormId}
      className="h-10 px-5 rounded-xl bg-[#264D3B] hover:bg-[#1B3B2B] text-[#F5EEDD] text-xs font-semibold shadow-sm transition-colors cursor-pointer"
    >
      {submitLabel}
    </button>
  </>
);
