import React from 'react';
import BrewAgentsPopout from './BrewAgentsPopout';
import BrewAgentCard, { modelShortName } from './BrewAgentCard';
import BrewSlotsPerServer from './BrewSlotsPerServer';
import { extractCodeBlock } from '../utils/codeExtractor';

export default function BrewConfigAgentsSection({
  roles,
  setRoles,
  models,
  selected,
  roleModels,
  engineModels,
  engine,
  showAgentsPopout,
  setShowAgentsPopout,
  selectAllRoles,
  clearAllRoles,
  toggleRole,
  setModel,
  setEditingAgent,
  agentStatuses,
  responses,
  agentErrors,
  lastMeta,
  setLeftPopout,
}) {
  return (
    <div className="brew-section brew-section--agents" style={{ flex: '1 1 0', overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <div className="brew-section-header brew-section-header--agents">
        <span className="brew-section-title">Agents</span>
        <div className="brew-agents-header-actions">
          <button type="button" className="brew-agents-bulk-btn" onClick={selectAllRoles} title="Select every agent role">All</button>
          <button type="button" className="brew-agents-bulk-btn" onClick={clearAllRoles} title="Clear agent selection">None</button>
          <button
            type="button"
            className={`brew-agents-popout-trigger${showAgentsPopout ? ' open' : ''}`}
            onClick={() => setShowAgentsPopout(v => !v)}
            aria-expanded={showAgentsPopout}
            title="Per-agent context and max token budgets"
          >BUDGETS</button>
        </div>
      </div>

      <BrewAgentsPopout
        open={showAgentsPopout}
        onClose={() => setShowAgentsPopout(false)}
        roles={roles}
        selected={selected}
        onRolesChange={setRoles}
      />

      <div className="brew-section-body" style={{ flex: '1 1 0', overflowY: 'auto', padding: '0.75rem' }}>
        <BrewSlotsPerServer
          roles={roles} selected={selected} roleModels={roleModels} setRoles={setRoles}
        />
        <div className="brew-agent-cards">
          {roles.map(role => {
            const isSelected   = selected.has(role.name);
            const modelPath    = roleModels[role.name] || '';
            const ctx          = role.context || 0;
            const launchStatus = agentStatuses?.get(role.name);
            const meta = launchStatus
              ? launchStatus.toUpperCase()
              : ctx > 0 ? `Context ${ctx.toLocaleString()}` : 'Context —';
            const response   = responses?.[role.name] ?? null;
            const agentError = agentErrors?.[role.name] ?? null;
            const timing     = lastMeta?.timings?.[role.name] ?? null;
            const hasResult  = !!(response || agentError);
            const onExpand   = hasResult ? () => {
              const { code, language } = response
                ? extractCodeBlock(response) : { code: null, language: null };
              const resultMeta = timing
                ? `${(timing.total_ms / 1000).toFixed(1)}s`
                : agentError ? 'FAILED' : meta;
              setLeftPopout({
                name: role.name.toUpperCase(), model: modelShortName(modelPath),
                meta: resultMeta, response, error: agentError,
                code: code && code.trim().length >= 10 ? code : null, language,
              });
            } : undefined;
            return (
              <BrewAgentCard
                key={role.name}
                name={role.name.toUpperCase()}
                modelPath={modelPath}
                meta={meta}
                selected={isSelected}
                hasResult={hasResult}
                onClick={() => toggleRole(role.name)}
                onEdit={() => setEditingAgent(role)}
                onExpand={onExpand}
                showCheckbox
                checked={isSelected}
                showModelSelect={engineModels.length > 0}
                models={engineModels.length > 0 ? engineModels : models}
                onModelChange={path => setModel(role.name, path)}
              />
            );
          })}
        </div>
      </div>
    </div>
  );
}
