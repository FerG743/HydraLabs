import { useCallback, useMemo } from 'react';
import {
  CheckCircle,
  XCircle,
  AlertCircle,
  Clock,
  Play,
  Pause,
  StopCircle,
  Info,
  AlertTriangle,
  Zap,
  Target
} from 'lucide-react';

// Custom hook for ResultCard logic (55 lines)
export const useResultCard = (result, index, onClick, options = {}) => {
  const { showAllFields = true, maxFields = 4 } = options;

  // Status icon and color mappings
  const STATUS_STYLES = {
    passed: { icon: CheckCircle, color: 'text-green-500' },
    success: { icon: CheckCircle, color: 'text-green-500' },
    completed: { icon: CheckCircle, color: 'text-green-500' },
    
    failed: { icon: XCircle, color: 'text-red-500' },
    error: { icon: XCircle, color: 'text-red-500' },
    rejected: { icon: XCircle, color: 'text-red-500' },
    
    warning: { icon: AlertTriangle, color: 'text-yellow-500' },
    partial: { icon: AlertTriangle, color: 'text-yellow-500' },
    
    running: { icon: Play, color: 'text-blue-500' },
    active: { icon: Play, color: 'text-blue-500' },
    processing: { icon: Play, color: 'text-blue-500' },
    
    pending: { icon: Clock, color: 'text-gray-500' },
    waiting: { icon: Clock, color: 'text-gray-500' },
    queued: { icon: Clock, color: 'text-gray-500' },
    
    paused: { icon: Pause, color: 'text-yellow-600' },
    stopped: { icon: StopCircle, color: 'text-gray-600' },
    cancelled: { icon: StopCircle, color: 'text-gray-600' },
    
    info: { icon: Info, color: 'text-blue-500' },
    default: { icon: Target, color: 'text-gray-500' }
  };

  // Get status styling
  const statusStyle = useMemo(() => 
    STATUS_STYLES[result.status] || STATUS_STYLES.default
  , [result.status]);

  const StatusIcon = statusStyle.icon;
  const statusColor = statusStyle.color;

  // Format title
  const title = useMemo(() => 
    result.name || result.title || result.description || 'Result'
  , [result.name, result.title, result.description]);

  // Format timestamp
  const formattedTimestamp = useMemo(() => {
    if (!result.timestamp) return null;
    
    const date = new Date(result.timestamp);
    if (isNaN(date.getTime())) return null;
    
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);
    
    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }, [result.timestamp]);

  // Extract and format display fields
  const displayFields = useMemo(() => {
    const excludedKeys = ['id', 'name', 'title', 'description', 'timestamp', 'status'];
    
    const entries = Object.entries(result).filter(([key]) => 
      !excludedKeys.includes(key) && result[key] != null
    );

    // Sort entries by importance/relevance
    const sortedEntries = entries.sort(([keyA], [keyB]) => {
      const importantFields = ['duration', 'tests', 'passed', 'failed', 'type', 'priority'];
      const aImportant = importantFields.indexOf(keyA);
      const bImportant = importantFields.indexOf(keyB);
      
      if (aImportant !== -1 && bImportant !== -1) return aImportant - bImportant;
      if (aImportant !== -1) return -1;
      if (bImportant !== -1) return 1;
      return keyA.localeCompare(keyB);
    });

    // Apply field limits
    if (!showAllFields && maxFields > 0) {
      return sortedEntries.slice(0, maxFields);
    }

    return sortedEntries;
  }, [result, showAllFields, maxFields]);

  // Click handler
  const handleClick = useCallback((e) => {
    if (onClick) {
      onClick(result, index, e);
    }
  }, [onClick, result, index]);

  // Format field values for better display
  const formatFieldValue = useCallback((key, value) => {
    // Duration formatting
    if (key.includes('duration') || key.includes('time')) {
      if (typeof value === 'number') {
        if (value > 1000) return `${(value / 1000).toFixed(1)}s`;
        return `${value}ms`;
      }
    }
    
    // Percentage formatting
    if (key.includes('rate') || key.includes('percent')) {
      if (typeof value === 'number') {
        return `${(value * 100).toFixed(1)}%`;
      }
    }

    // Count formatting
    if (['tests', 'passed', 'failed', 'total'].includes(key)) {
      if (typeof value === 'number') {
        if (value >= 1000) return `${(value / 1000).toFixed(1)}k`;
        return value.toString();
      }
    }

    return String(value);
  }, []);

  // Apply formatting to display fields
  const formattedDisplayFields = useMemo(() => 
    displayFields.map(([key, value]) => [key, formatFieldValue(key, value)])
  , [displayFields, formatFieldValue]);

  return {
    StatusIcon,
    statusColor,
    title,
    formattedTimestamp,
    displayFields: formattedDisplayFields,
    handleClick
  };
};