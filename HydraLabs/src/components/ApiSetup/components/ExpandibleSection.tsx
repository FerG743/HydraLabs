import React from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';

const ExpandableSection = ({
  title,
  icon: Icon,
  isExpanded,
  onToggle,
  badge,
  subtitle,
  children,
  className = '',
  headerClassName = '',
  contentClassName = ''
}) => {
  return (
    <div className={`border-b border-gray-100 ${className}`}>
      <button
        onClick={onToggle}
        className={`w-full flex items-center gap-3 p-3 text-left hover:bg-gray-50 transition-all duration-200 ${headerClassName}`}
      >
        {isExpanded ? (
          <ChevronDown className="w-4 h-4 flex-shrink-0" />
        ) : (
          <ChevronRight className="w-4 h-4 flex-shrink-0" />
        )}
        
        {Icon && <Icon className="w-4 h-4 text-gray-600 flex-shrink-0" />}
        
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium truncate">{title}</span>
            {badge && (
              <span className="text-xs bg-gray-100 text-gray-600 px-2 py-0.5 rounded flex-shrink-0">
                {badge}
              </span>
            )}
          </div>
          {subtitle && (
            <div className="text-xs text-gray-500 mt-0.5 truncate">
              {subtitle}
            </div>
          )}
        </div>
      </button>
      
      {isExpanded && (
        <div className={`p-4 pt-0 animate-slide-down ${contentClassName}`}>
          {children}
        </div>
      )}
    </div>
  );
};

export default ExpandableSection;