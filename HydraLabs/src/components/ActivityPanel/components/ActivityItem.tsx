import React from 'react';
import { Copy } from 'lucide-react';
import { useActivityItem } from './hooks/useActivityItem';

// ActivityItem Component (40 lines)
const ActivityItem = ({
  activity,
  index = 0,
  onClick,
  className = '',
  showCopyButton = true,
  showTimestamp = true,
  animationDelay = 50
}) => {
  const {
    statusIcon: StatusIcon,
    typeIcon: TypeIcon, 
    statusColor,
    formattedTimestamp,
    handleClick,
    handleCopy
  } = useActivityItem(activity, index, onClick);

  return (
    <div
      className={`flex items-start gap-3 p-2 hover:bg-gray-50 rounded group cursor-pointer 
        animate-fade-in-up transition-all duration-200 ${className}`}
      style={{ animationDelay: `${index * animationDelay}ms` }}
      onClick={handleClick}
    >
      {/* Status and Type Icons */}
      <div className="flex items-center gap-2 min-w-0">
        <StatusIcon className={`w-4 h-4 ${statusColor}`} />
        <TypeIcon className="w-4 h-4 text-gray-500" />
      </div>

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between">
          <p className="text-sm font-medium text-gray-900 truncate">
            {activity.message || activity.title || 'No message'}
          </p>
          {showTimestamp && (
            <span className="text-xs text-gray-500 ml-2 flex-shrink-0">
              {formattedTimestamp}
            </span>
          )}
        </div>
        
        {/* Activity Details */}
        {activity.details && (
          <div className="mt-1 text-xs text-gray-600">
            {typeof activity.details === 'string' 
              ? activity.details 
              : `${Object.keys(activity.details).length} details`
            }
          </div>
        )}
      </div>

      {/* Copy Button */}
      {showCopyButton && (
        <button
          onClick={handleCopy}
          className="opacity-0 group-hover:opacity-100 p-1 text-gray-400 hover:text-gray-600 
            transition-all duration-200 hover:scale-110 active:scale-95"
          title="Copy activity message"
        >
          <Copy className="w-3 h-3" />
        </button>
      )}
    </div>
  );
};

export default ActivityItem;