import { useCallback, useMemo } from 'react';
import {
  BarChart,
  Clock,
  Zap,
  Database,
  Users,
  Target,
  Globe,
  CheckCircle,
  XCircle,
  TrendingUp,
  TrendingDown,
  Activity,
  Server,
  Shield,
  FileText,
  Settings,
  Eye,
  Download,
  Upload
} from 'lucide-react';

// Custom hook for MetricCard logic (50 lines)
export const useMetricCard = (metric, onClick) => {
  // Icon mapping for different metric types
  const METRIC_ICONS = {
    // Performance metrics
    responsetime: Clock,
    avgresponsetime: Clock,
    duration: Clock,
    time: Clock,
    latency: Clock,
    
    // Count metrics
    total: BarChart,
    count: BarChart,
    requests: Globe,
    users: Users,
    sessions: Users,
    
    // Success/Error metrics
    passed: CheckCircle,
    failed: XCircle,
    success: CheckCircle,
    errors: XCircle,
    
    // Rate metrics
    throughput: TrendingUp,
    rate: TrendingUp,
    percentage: BarChart,
    ratio: BarChart,
    
    // System metrics
    cpu: Server,
    memory: Database,
    disk: Database,
    network: Globe,
    
    // Security metrics
    vulnerabilities: Shield,
    threats: Shield,
    security: Shield,
    
    // File metrics
    size: FileText,
    files: FileText,
    documents: FileText,
    
    // Default
    default: Activity
  };

  // Extract key and value from metric (supports both array and object formats)
  const [key, value] = useMemo(() => {
    if (Array.isArray(metric)) {
      return metric;
    }
    if (typeof metric === 'object' && metric !== null) {
      const entries = Object.entries(metric);
      return entries.length > 0 ? entries[0] : ['unknown', 'N/A'];
    }
    return ['metric', String(metric)];
  }, [metric]);

  // Get appropriate icon for the metric key
  const MetricIcon = useMemo(() => {
    const iconKey = key.toLowerCase().replace(/[^a-z]/g, '');
    return METRIC_ICONS[iconKey] || METRIC_ICONS.default;
  }, [key]);

  // Format the metric key for display
  const formattedKey = useMemo(() => {
    return key
      .replace(/([A-Z])/g, ' $1') // Add space before capitals
      .replace(/[-_]/g, ' ') // Replace dashes/underscores with spaces
      .toLowerCase()
      .replace(/\b\w/g, l => l.toUpperCase()) // Capitalize first letter of each word
      .trim();
  }, [key]);

  // Format the value for display
  const formattedValue = useMemo(() => {
    if (typeof value === 'number') {
      // Format numbers with appropriate precision
      if (value >= 1000000) {
        return `${(value / 1000000).toFixed(1)}M`;
      }
      if (value >= 1000) {
        return `${(value / 1000).toFixed(1)}K`;
      }
      if (value % 1 !== 0) {
        return value.toFixed(2);
      }
      return value.toString();
    }
    
    if (typeof value === 'string') {
      // Handle percentage strings
      if (value.endsWith('%')) {
        return value;
      }
      // Handle time strings
      if (value.includes('ms') || value.includes('s')) {
        return value;
      }
    }
    
    return String(value);
  }, [value]);

  // Click handler
  const handleClick = useCallback(() => {
    if (onClick) {
      onClick(metric, key, value);
    }
  }, [onClick, metric, key, value]);

  return {
    key,
    value,
    formattedKey,
    formattedValue,
    MetricIcon,
    handleClick
  };
};