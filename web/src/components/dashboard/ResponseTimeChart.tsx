import { useMemo } from 'react';
import { Area, AreaChart, YAxis } from 'recharts';
import type { ChartConfig } from '@/components/ui/chart';
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/ui/chart';
import type { CheckStatus } from '@/types';

interface ResponseTimeChartProps {
  history: CheckStatus[];
  isUp?: boolean;
  height?: string;
}

const chartConfig = {
  responseTime: {
    label: 'Response Time',
    color: 'var(--t-green)',
  },
} satisfies ChartConfig;

const chartConfigDown = {
  responseTime: {
    label: 'Response Time',
    color: 'var(--t-red)',
  },
} satisfies ChartConfig;

export function ResponseTimeChart({
  history,
  isUp = true,
  height = 'h-24',
}: ResponseTimeChartProps) {
  const chartData = useMemo(() => {
    if (!history || history.length === 0) return [];
    
    return [...history].reverse().map((h) => ({
      date: h.checked_at,
      responseTime: h.response_time_ms || 0,
    }));
  }, [history]);

  if (!history || history.length === 0) {
    return (
      <div className={`${height} w-full flex items-center justify-center text-terminal-muted text-sm`}>
        No data available
      </div>
    );
  }

  const config = isUp ? chartConfig : chartConfigDown;
  const strokeColor = isUp ? 'rgb(var(--t-green))' : 'rgb(var(--t-red))';

  return (
    <ChartContainer config={config} className={`${height} w-full`}>
      <AreaChart data={chartData} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
        <defs>
          <linearGradient id={`fillResponseTime-${isUp ? 'up' : 'down'}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor={strokeColor} stopOpacity={0.3} />
            <stop offset="95%" stopColor={strokeColor} stopOpacity={0.05} />
          </linearGradient>
        </defs>
        <YAxis domain={['dataMin', 'dataMax']} hide />
        <ChartTooltip
          cursor={false}
          content={
            <ChartTooltipContent
              className="bg-terminal-surface border-terminal-border text-terminal-text"
              labelFormatter={(_, payload) => {
                if (!payload?.[0]?.payload?.date) return '';
                const date = new Date(payload[0].payload.date);
                return date.toLocaleString();
              }}
              formatter={(value) => (
                <span className="text-terminal-text font-mono">
                  {value} <span className="text-terminal-muted">ms</span>
                </span>
              )}
              hideIndicator
              hideLabel={false}
            />
          }
        />
        <Area
          dataKey="responseTime"
          type="monotone"
          fill={`url(#fillResponseTime-${isUp ? 'up' : 'down'})`}
          stroke={strokeColor}
          strokeWidth={1.5}
        />
      </AreaChart>
    </ChartContainer>
  );
}
