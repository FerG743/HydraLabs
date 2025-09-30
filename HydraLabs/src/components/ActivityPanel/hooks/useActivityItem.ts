import { useCallback, useMemo } from 'react';
import { 
  CheckCircle, 
  XCircle, 
  AlertCircle, 
  Info,
  Globe,
  Target,
  Shield,
  Zap,
  Activity,
  Bug,
  Users,
  Settings,
  Database,
  Code
} from 'lucide-react';

// Custom hook for ActivityItem logic (45 lines)
export const useActivityItem = (activity, index, onClick) => {
  // Icon mappings
  const STATUS_ICONS = {
    success: CheckCircle,
    passed: CheckCircle,
    error: XCircle,
    failed: XCircle,
    warning: AlertCircle,
    info: Info,
    pending: AlertCircle
  };

  const TYPE_ICONS = {
    request: Globe,
    test: Target,
    validation: Shield,
    performance: Zap,
    error: Bug,
    info: Info,
    user: Users,
    system: Settings,
    api: Database,
    security: Shield,
    code: Code,
    default: Activity
  };

  const STATUS_COLORS = {
    success: 'text-green-500',
    passed: 'text-green-500',
    error: 'text-red-500',
    failed: 'text-red-500',
    warning: 'text-yellow-500',
    info: 'text-primary',
    pending: 'text-muted-foreground'
  };

  // Memoized icon and color calculations
  const statusIcon = useMemo(() => 
    STATUS_ICONS[activity.status] || STATUS_ICONS.info
  , [activity.status]);

  const typeIcon = useMemo(() => 
    TYPE_ICONS[activity.type] || TYPE_ICONS.default
  , [activity.type]);

  const statusColor = useMemo(() => 
    STATUS_COLORS[activity.status] || STATUS_COLORS.info
  , [activity.status]);

  // Format timestamp
  const formattedTimestamp = useMemo(() => {
    if (!activity.timestamp) return '';
    
    const date = new Date(activity.timestamp);
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
    });
  }, [activity.timestamp]);

  // Event handlers
  const handleClick = useCallback((e) => {
    if (onClick) {
      onClick(activity, index, e);
    }
  }, [activity, index, onClick]);

  const handleCopy = useCallback((e) => {
    e.stopPropagation();
    
    const textToCopy = activity.message || JSON.stringify(activity, null, 2);
    
    if (navigator.clipboard) {
      navigator.clipboard.writeText(textToCopy).catch(err => {
        console.warn('Failed to copy to clipboard:', err);
        // Fallback for older browsers
        fallbackCopy(textToCopy);
      });
    } else {
      fallbackCopy(textToCopy);
    }
  }, [activity]);

  // Fallback copy method for older browsers
  const fallbackCopy = (text) => {
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.opacity = '0';
    document.body.appendChild(textArea);
    textArea.select();
    
    try {
      document.execCommand('copy');
    } catch (err) {
      console.warn('Fallback copy failed:', err);
    }
    
    document.body.removeChild(textArea);
  };

  return {
    statusIcon,
    typeIcon,
    statusColor,
    formattedTimestamp,
    handleClick,
    handleCopy
  };
};