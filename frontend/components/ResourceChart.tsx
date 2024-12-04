'use client'

import { useEffect, useState, useMemo } from 'react';
import { 
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer
} from 'recharts';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Cpu, MemoryStick, HardDrive, Network } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Skeleton } from '@/components/ui/skeleton';
import { getVPSMetrics } from '@/app/actions';
import { formatBytes, formatBitrate } from '@/utils/formatUtils';

interface ResourceMonitoringProps {
  vpsId: string;
  isRunning: boolean;
}

interface MetricData {
  time: string;
  cpu: {
    usage: number;
  };
  memory: {
    used: number;
    total: number;
    cache: number;
  };
  disk: {
    read_bytes: number;
    write_bytes: number;
    read_ops: number;
    write_ops: number;
    read_speed: number;
    write_speed: number;
  };
  network: {
    rx_bytes: number;
    tx_bytes: number;
    rx_packets: number;
    tx_packets: number;
    rx_speed: number;
    tx_speed: number;
  };
}

interface FlattenedMetric {
  time: string;
  cpuUsage: number;
  memoryUsed: number;
  diskReadSpeed: number;
  diskWriteSpeed: number;
  networkRxSpeed: number;
  networkTxSpeed: number;
}

const ChartTooltip = ({ active, payload, label, valueFormatter }) => {
  if (!active || !payload) return null;
  
  return (
    <div className="rounded-lg bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/85 p-2 shadow-lg border border-border">
      <p className="text-sm font-medium text-foreground">
        {new Date(label).toLocaleTimeString()}
      </p>
      {payload.map((entry, index) => (
        <p key={index} className="text-sm" style={{ color: entry.color }}>
          {`${entry.name}: ${valueFormatter(entry.value)}`}
        </p>
      ))}
    </div>
  );
};

export default function ResourceMonitoring({ vpsId, isRunning }: ResourceMonitoringProps) {
  const [metrics, setMetrics] = useState<MetricData[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const recentMetrics = useMemo(() => {
    return metrics.slice(-30).map((metric, index, array) => {
      const smoothingFactor = 0.3;
      const previousMetric = array[index - 1];
      
      // Smooth out the values using exponential smoothing
      const smoothValue = (current: number, previous: number) => 
        previousMetric ? current * smoothingFactor + previous * (1 - smoothingFactor) : current;

      return {
        time: metric.time,
        cpuUsage: smoothValue(metric.cpu.usage * 100, previousMetric?.cpu.usage * 100),
        memoryUsed: smoothValue(metric.memory.used, previousMetric?.memory.used),
        diskReadSpeed: smoothValue(metric.disk.read_speed, previousMetric?.disk.read_speed),
        diskWriteSpeed: smoothValue(metric.disk.write_speed, previousMetric?.disk.write_speed),
        networkRxSpeed: smoothValue(metric.network.rx_speed, previousMetric?.network.rx_speed),
        networkTxSpeed: smoothValue(metric.network.tx_speed, previousMetric?.network.tx_speed),
      };
    });
  }, [metrics]);

  useEffect(() => {
    if (!isRunning) {
      setMetrics([]);
      setLoading(false);
      return;
    }

    let mounted = true;

    const fetchMetrics = async () => {
      try {
        const data = await getVPSMetrics(vpsId);
        if (mounted) {
          setMetrics(prev => {
            // Ensure smooth transition by maintaining some history
            const newData = [...prev.slice(-25), ...data.slice(-5)];
            return newData;
          });
          setError(null);
        }
      } catch (err) {
        if (mounted) {
          console.error('Failed to fetch metrics:', err);
          setError('Failed to fetch resource metrics');
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    };

    fetchMetrics();
    const interval = setInterval(fetchMetrics, 2000);
    
    return () => {
      mounted = false;
      clearInterval(interval);
    };
  }, [vpsId, isRunning]);

  if (!isRunning) {
    return null;
  }

  if (loading) {
    return (
      <div className="grid gap-6 lg:grid-cols-2">
        {[1, 2, 3, 4].map((i) => (
          <Card key={i} className="transition-all duration-200 hover:shadow-lg">
            <CardHeader>
              <CardTitle>
                <Skeleton className="h-6 w-32" />
              </CardTitle>
            </CardHeader>
            <CardContent>
              <Skeleton className="h-64 w-full" />
            </CardContent>
          </Card>
        ))}
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive" className="animate-in fade-in slide-in-from-top-2">
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  const chartCommonProps = {
    margin: { top: 5, right: 5, left: 5, bottom: 5 },
    className: "transition-all duration-200",
  };

  const lineCommonProps = {
    strokeWidth: 2,
    dot: false,
    isAnimationActive: true,
    animationDuration: 300,
  };

  return (
    <div className="grid gap-6 lg:grid-cols-2">
      <Card className="transition-all duration-200 hover:shadow-lg">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Cpu className="h-5 w-5 text-blue-500" />
            CPU Usage
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={recentMetrics} {...chartCommonProps}>
                <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
                <XAxis 
                  dataKey="time"
                  tickFormatter={(time) => new Date(time).toLocaleTimeString()}
                  stroke="currentColor"
                  opacity={0.5}
                />
                <YAxis 
                  domain={[0, 100]} 
                  tickFormatter={(value) => `${value}%`}
                  stroke="currentColor"
                  opacity={0.5}
                />
                <Tooltip content={<ChartTooltip valueFormatter={(value) => `${value.toFixed(1)}%`} />} />
                <Line 
                  type="monotone"
                  dataKey="cpuUsage"
                  stroke="#3b82f6"
                  {...lineCommonProps}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>

      <Card className="transition-all duration-200 hover:shadow-lg">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <MemoryStick className="h-5 w-5 text-purple-500" />
            Memory Usage
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={recentMetrics} {...chartCommonProps}>
                <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
                <XAxis 
                  dataKey="time"
                  tickFormatter={(time) => new Date(time).toLocaleTimeString()}
                  stroke="currentColor"
                  opacity={0.5}
                />
                <YAxis 
                  tickFormatter={formatBytes}
                  stroke="currentColor"
                  opacity={0.5}
                />
                <Tooltip content={<ChartTooltip valueFormatter={formatBytes} />} />
                <defs>
                  <linearGradient id="memoryGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.4}/>
                    <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <Area
                  type="monotone"
                  dataKey="memoryUsed"
                  stroke="#8b5cf6"
                  fill="url(#memoryGradient)"
                  isAnimationActive={true}
                  animationDuration={300}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>

      <Card className="transition-all duration-200 hover:shadow-lg">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <HardDrive className="h-5 w-5 text-green-500" />
            Disk I/O
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={recentMetrics} {...chartCommonProps}>
                <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
                <XAxis 
                  dataKey="time"
                  tickFormatter={(time) => new Date(time).toLocaleTimeString()}
                  stroke="currentColor"
                  opacity={0.5}
                />
                <YAxis 
                  tickFormatter={(value) => formatBytes(value) + '/s'}
                  stroke="currentColor"
                  opacity={0.5}
                />
                <Tooltip content={<ChartTooltip valueFormatter={(value) => formatBytes(value) + '/s'} />} />
                <Line 
                  type="monotone"
                  name="Read"
                  dataKey="diskReadSpeed"
                  stroke="#22c55e"
                  {...lineCommonProps}
                />
                <Line 
                  type="monotone"
                  name="Write"
                  dataKey="diskWriteSpeed"
                  stroke="#15803d"
                  {...lineCommonProps}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>

      <Card className="transition-all duration-200 hover:shadow-lg">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Network className="h-5 w-5 text-orange-500" />
            Network Traffic
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={recentMetrics} {...chartCommonProps}>
                <CartesianGrid strokeDasharray="3 3" opacity={0.2} />
                <XAxis 
                  dataKey="time"
                  tickFormatter={(time) => new Date(time).toLocaleTimeString()}
                  stroke="currentColor"
                  opacity={0.5}
                />
                <YAxis 
                  tickFormatter={formatBitrate}
                  stroke="currentColor"
                  opacity={0.5}
                />
                <Tooltip content={<ChartTooltip valueFormatter={formatBitrate} />} />
                <Line 
                  type="monotone"
                  name="Download"
                  dataKey="networkRxSpeed"
                  stroke="#f97316"
                  {...lineCommonProps}
                />
                <Line 
                  type="monotone"
                  name="Upload"
                  dataKey="networkTxSpeed"
                  stroke="#ea580c"
                  {...lineCommonProps}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}