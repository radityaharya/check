
import { useState } from 'react';
import { ChevronDown, Edit3, Play, Pause, Trash2, Clock } from 'lucide-react';
import { cn } from '@/lib/utils';
import { StatusDot, StatusBar } from '@/components/ui/status-bar';
import { Skeleton } from '@/components/ui/spinner';
import { ResponseTimeChart } from './ResponseTimeChart';
import { useTimeAgoTick, formatTimeAgo } from '@/hooks/use-time-ago';
import {
  getCheckTarget,
  getCheckTypeClass,
  formatResponseTime,
  formatDate,
} from '@/lib/helpers';
import type { Check, CheckGroup, Group, Tag, TimeRange } from '@/types';

interface MonitorsListProps {
  groups: CheckGroup[];
  isLoading: boolean;
  expandedGroups: string[];
  onToggleGroup: (groupId: string) => void;
  selectedCheckId?: number;
  onSelectCheck: (check: Check) => void;
  onEditCheck: (check: Check) => void;
  onDeleteCheck: (check: Check) => void;
  onTriggerCheck: (check: Check) => void;
  onToggleEnabled: (check: Check) => void;
  onShowHistory: (check: Check) => void;
  onEditGroup: (group: Group) => void;
  onEditTag: (tag: Tag) => void;
  timeRange: TimeRange;
}

export function MonitorsList({
  groups,
  isLoading,
  expandedGroups,
  onToggleGroup,
  selectedCheckId,
  onSelectCheck,
  onEditCheck,
  onDeleteCheck,
  onTriggerCheck,
  onToggleEnabled,
  onShowHistory,
  onEditGroup,
  onEditTag,
  timeRange,
}: MonitorsListProps) {
  // Re-render every 5 seconds to keep "time ago" displays fresh
  useTimeAgoTick(5000);

  if (isLoading) {
    return (
      <div className="bg-terminal-surface border border-terminal-border rounded-sm overflow-hidden">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="border-b border-terminal-border last:border-b-0">
            <div className="flex items-center gap-4 p-4 border-b border-terminal-border">
              <div className="w-1 h-8 rounded skeleton" />
              <div className="w-2.5 h-2.5 rounded-full skeleton" />
              <div className="flex-grow">
                <Skeleton className="h-4 w-32" />
              </div>
              <Skeleton className="h-5 w-5" />
            </div>
            <div className="divide-y divide-terminal-border">
              {Array.from({ length: 2 }).map((_, j) => (
                <div key={j} className="p-4">
                  <div className="flex items-center gap-4">
                    <div className="w-2.5 h-2.5 rounded-full skeleton" />
                    <div className="flex-grow">
                      <Skeleton className="h-4 w-40 mb-2" />
                      <Skeleton className="h-3 w-64" />
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (groups.length === 0) {
    return (
      <div className="text-center py-20 border border-dashed border-terminal-border rounded-sm">
        <div className="text-terminal-muted mb-2 text-4xl">_</div>
        <div className="text-terminal-muted mb-4 text-sm">
          no monitors configured
        </div>
      </div>
    );
  }

  return (
    <div className="bg-terminal-surface border border-terminal-border rounded-sm overflow-hidden">
      {groups.map((group) => (
        <GroupItem
          key={group.id}
          group={group}
          isExpanded={expandedGroups.includes(String(group.id))}
          onToggle={() => onToggleGroup(String(group.id))}
          selectedCheckId={selectedCheckId}
          onSelectCheck={onSelectCheck}
          onEditCheck={onEditCheck}
          onDeleteCheck={onDeleteCheck}
          onTriggerCheck={onTriggerCheck}
          onToggleEnabled={onToggleEnabled}
          onShowHistory={onShowHistory}
          onEditGroup={onEditGroup}
          onEditTag={onEditTag}
          timeRange={timeRange}
        />
      ))}
    </div>
  );
}

interface GroupItemProps {
  group: CheckGroup;
  isExpanded: boolean;
  onToggle: () => void;
  selectedCheckId?: number;
  onSelectCheck: (check: Check) => void;
  onEditCheck: (check: Check) => void;
  onDeleteCheck: (check: Check) => void;
  onTriggerCheck: (check: Check) => void;
  onToggleEnabled: (check: Check) => void;
  onShowHistory: (check: Check) => void;
  onEditGroup: (group: Group) => void;
  onEditTag: (tag: Tag) => void;
  timeRange: TimeRange;
}

function GroupItem({
  group,
  isExpanded,
  onToggle,
  selectedCheckId,
  onSelectCheck,
  onEditCheck,
  onDeleteCheck,
  onTriggerCheck,
  onToggleEnabled,
  onShowHistory,
  onEditGroup,
  onEditTag,
  timeRange,
}: GroupItemProps) {
  return (
    <div className="border-b border-terminal-border last:border-b-0">
      {/* Group Header */}
      <button
        onClick={onToggle}
        className="w-full flex items-center gap-4 p-4 hover:bg-terminal-border/30 transition group"
      >
        <div
          className={cn(
            'w-1 h-8 rounded',
            group.is_up ? 'bg-terminal-green' : 'bg-terminal-red'
          )}
        />
        <StatusDot
          isUp={group.is_up}
          enabled={true}
          glow={true}
        />
        <div className="flex-grow text-left">
          <div className="flex items-center gap-3">
            <span className="font-semibold">{group.name}</span>
            <span className="text-terminal-muted text-xs">
              {group.up_count}/{group.checks.length}
            </span>
          </div>
        </div>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onEditGroup(group);
          }}
          className="text-terminal-muted hover:text-terminal-text opacity-0 group-hover:opacity-100 transition p-1"
          title="Edit group"
        >
          <Edit3 className="w-4 h-4" />
        </button>
        <ChevronDown
          className={cn(
            'w-5 h-5 text-terminal-muted transition-transform',
            isExpanded && 'rotate-180'
          )}
        />
      </button>

      {/* Checks List */}
      {isExpanded && (
        <ChecksList
          checks={group.checks}
          selectedCheckId={selectedCheckId}
          onSelectCheck={onSelectCheck}
          onEditCheck={onEditCheck}
          onDeleteCheck={onDeleteCheck}
          onTriggerCheck={onTriggerCheck}
          onToggleEnabled={onToggleEnabled}
          onShowHistory={onShowHistory}
          onEditTag={onEditTag}
          timeRange={timeRange}
        />
      )}
    </div>
  );
}

interface ChecksListProps {
  checks: Check[];
  selectedCheckId?: number;
  onSelectCheck: (check: Check) => void;
  onEditCheck: (check: Check) => void;
  onDeleteCheck: (check: Check) => void;
  onTriggerCheck: (check: Check) => void;
  onToggleEnabled: (check: Check) => void;
  onShowHistory: (check: Check) => void;
  onEditTag: (tag: Tag) => void;
  timeRange: TimeRange;
}

function ChecksList({
  checks,
  selectedCheckId,
  onSelectCheck,
  onEditCheck,
  onDeleteCheck,
  onTriggerCheck,
  onToggleEnabled,
  onShowHistory,
  onEditTag,
  timeRange,
}: ChecksListProps) {
  const [expandedCheckId, setExpandedCheckId] = useState<number | null>(null);

  const handleToggleExpand = (checkId: number) => {
    setExpandedCheckId(expandedCheckId === checkId ? null : checkId);
  };

  return (
    <div className="divide-y divide-terminal-border">
      {checks.map((check) => (
        <CheckItem
          key={check.id}
          check={check}
          isSelected={selectedCheckId === check.id}
          isExpanded={expandedCheckId === check.id}
          onSelect={() => onSelectCheck(check)}
          onToggleExpand={() => handleToggleExpand(check.id)}
          onEdit={() => onEditCheck(check)}
          onDelete={() => onDeleteCheck(check)}
          onTrigger={() => onTriggerCheck(check)}
          onToggleEnabled={() => onToggleEnabled(check)}
          onShowHistory={() => onShowHistory(check)}
          onEditTag={onEditTag}
          timeRange={timeRange}
        />
      ))}
    </div>
  );
}

interface CheckItemProps {
  check: Check;
  isSelected: boolean;
  isExpanded: boolean;
  onSelect: () => void;
  onToggleExpand: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onTrigger: () => void;
  onToggleEnabled: () => void;
  onShowHistory: () => void;
  onEditTag: (tag: Tag) => void;
  timeRange: TimeRange;
}

function CheckItem({
  check,
  isSelected,
  isExpanded,
  onSelect,
  onToggleExpand,
  onEdit,
  onDelete,
  onTrigger,
  onToggleEnabled,
  onShowHistory,
  onEditTag,
  timeRange,
}: CheckItemProps) {
  return (
    <div
      className={cn(
        'transition border-l-2 border-transparent',
        isSelected && 'lg:bg-terminal-green/5 lg:border-l-terminal-green'
      )}
    >
      {/* Main row - clickable (only selects on desktop, toggles expand on mobile) */}
      <div
        onClick={() => {
          // On mobile (< lg), toggle expand instead of selecting
          if (window.innerWidth < 1024) {
            onToggleExpand();
          } else {
            onSelect();
          }
        }}
        className="p-4 cursor-pointer group hover:bg-terminal-border/30"
      >
        <div className="flex items-center gap-4">
          <StatusDot
            isUp={check.is_up ?? true}
            enabled={check.enabled}
            glow={check.enabled}
            pulse={check.enabled && check.is_up !== false}
          />
          <div className="flex-grow min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span
                className={cn(
                  'font-semibold truncate',
                  !check.enabled && 'text-terminal-muted'
                )}
              >
                {check.name}
              </span>
              <span
                className={cn(
                  'text-[10px] px-1.5 py-0.5 rounded uppercase',
                  getCheckTypeClass(check.type)
                )}
              >
                {check.type}
              </span>
              {check.tags?.map((tag) => (
                <button
                  key={tag.id}
                  onClick={(e) => {
                    e.stopPropagation();
                    onEditTag(tag);
                  }}
                  className="text-[10px] px-1.5 py-0.5 rounded hover:opacity-80"
                  style={{
                    background: `${tag.color}30`,
                    color: tag.color,
                  }}
                >
                  {tag.name}
                </button>
              ))}
            </div>
            <div className="text-xs text-terminal-muted truncate mt-1">
              {getCheckTarget(check)}
            </div>
          </div>

          {/* Desktop Actions - hidden on mobile */}
          <div className="hidden lg:flex items-center gap-1 opacity-0 group-hover:opacity-100 transition">
            <button
              onClick={(e) => {
                e.stopPropagation();
                onTrigger();
              }}
              className="p-1.5 text-terminal-muted hover:text-terminal-cyan rounded hover:bg-terminal-border"
              title="Trigger check now"
            >
              <Play className="w-4 h-4" />
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onToggleEnabled();
              }}
              className="p-1.5 text-terminal-muted hover:text-terminal-yellow rounded hover:bg-terminal-border"
              title={check.enabled ? 'Pause' : 'Resume'}
            >
              {check.enabled ? (
                <Pause className="w-4 h-4" />
              ) : (
                <Play className="w-4 h-4" />
              )}
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onShowHistory();
              }}
              className="p-1.5 text-terminal-muted hover:text-terminal-purple rounded hover:bg-terminal-border"
              title="View history"
            >
              <Clock className="w-4 h-4" />
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onEdit();
              }}
              className="p-1.5 text-terminal-muted hover:text-terminal-text rounded hover:bg-terminal-border"
              title="Edit"
            >
              <Edit3 className="w-4 h-4" />
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onDelete();
              }}
              className="p-1.5 text-terminal-muted hover:text-terminal-red rounded hover:bg-terminal-border"
              title="Delete"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>

          {/* Response Time */}
          <div className="hidden sm:block text-xs text-terminal-muted">
            {formatResponseTime(check.last_status?.response_time_ms)}
          </div>

          {/* Last Checked */}
          <div 
            className="hidden sm:block text-xs text-terminal-muted min-w-[60px] text-right"
            title={
              check.last_status?.region 
                ? `Region: ${check.last_status.region}${check.last_checked_at ? `\nChecked: ${formatDate(check.last_checked_at)}` : ''}`
                : check.last_checked_at 
                  ? formatDate(check.last_checked_at)
                  : undefined
            }
          >
            {formatTimeAgo(check.last_checked_at)}
          </div>



          {/* Mobile expand button - visible only on mobile */}
          <button
            onClick={(e) => {
              e.stopPropagation();
              onToggleExpand();
            }}
            className="lg:hidden p-1 text-terminal-muted hover:text-terminal-text"
            title={isExpanded ? 'Collapse' : 'Expand'}
          >
            <ChevronDown
              className={cn(
                'w-5 h-5 transition-transform',
                isExpanded && 'rotate-180'
              )}
            />
          </button>
        </div>

        {/* Status Bar - always visible */}
        {check.history && check.history.length > 0 && (
          <div className="mt-3">
            <StatusBar
              history={check.history}
              check={check}
              timeRange={timeRange}
            />
          </div>
        )}
      </div>

      {/* Mobile Accordion - expanded details (hidden on lg screens) */}
      {isExpanded && (
        <div className="lg:hidden border-t border-terminal-border bg-terminal-bg/50">
          <div className="p-5">
            {/* Response Time Chart */}
            {check.history && check.history.length > 0 && (
              <div className="mb-4">
                <div className="flex justify-between items-center text-xs mb-2">
                  <span className="text-terminal-muted">response time history</span>
                  <span className="text-terminal-muted">
                    interval: <span className="text-terminal-text">{check.interval_seconds}s</span>
                  </span>
                </div>
                <ResponseTimeChart
                  history={check.history}
                  isUp={check.is_up}
                  height="h-24"
                />
              </div>
            )}

            {/* Quick Stats */}
            <div className="grid grid-cols-2 gap-3 mb-4 text-sm">
              <div className="bg-terminal-surface border border-terminal-border rounded p-2">
                <div className="text-[10px] text-terminal-muted uppercase">Latency</div>
                <div className="font-bold">{formatResponseTime(check.last_status?.response_time_ms)}</div>
              </div>
              <div className="bg-terminal-surface border border-terminal-border rounded p-2">
                <div className="text-[10px] text-terminal-muted uppercase">Last Check</div>
                <div 
                  className="font-bold"
                  title={
                    check.last_status?.region 
                      ? `Region: ${check.last_status.region}${check.last_checked_at ? `\nChecked: ${formatDate(check.last_checked_at)}` : ''}`
                      : check.last_checked_at 
                        ? formatDate(check.last_checked_at)
                        : undefined
                  }
                >
                  {formatTimeAgo(check.last_checked_at)}
                </div>
              </div>
            </div>

            {/* Action Buttons */}
            <div className="flex gap-2 flex-wrap">
              <button
                onClick={onSelect}
                className="text-[10px] bg-terminal-cyan/20 hover:bg-terminal-cyan/30 text-terminal-cyan px-3 py-1.5 rounded uppercase tracking-wide font-bold"
              >
                view details
              </button>
              <button
                onClick={onTrigger}
                disabled={!check.enabled}
                className="text-[10px] bg-terminal-green/20 hover:bg-terminal-green/30 text-terminal-green px-3 py-1.5 rounded uppercase tracking-wide disabled:opacity-50"
              >
                check now
              </button>
              <button
                onClick={onToggleEnabled}
                className={cn(
                  'text-[10px] px-3 py-1.5 rounded uppercase tracking-wide',
                  check.enabled
                    ? 'bg-terminal-yellow/20 hover:bg-terminal-yellow/30 text-terminal-yellow'
                    : 'bg-terminal-green/20 hover:bg-terminal-green/30 text-terminal-green'
                )}
              >
                {check.enabled ? 'pause' : 'resume'}
              </button>
              <button
                onClick={onEdit}
                className="text-[10px] bg-terminal-border hover:bg-terminal-muted text-terminal-text px-3 py-1.5 rounded uppercase tracking-wide"
              >
                edit
              </button>
              <button
                onClick={onShowHistory}
                className="text-[10px] bg-terminal-border hover:bg-terminal-muted text-terminal-text px-3 py-1.5 rounded uppercase tracking-wide"
              >
                changes log
              </button>
              <button
                onClick={onDelete}
                className="text-[10px] bg-terminal-red/20 hover:bg-terminal-red/30 text-terminal-red px-3 py-1.5 rounded uppercase tracking-wide"
              >
                delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
