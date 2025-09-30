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
        className={`w-full flex items-center gap-3 p-3 text-left hover:bg-muted transition-all duration-200 ${headerClassName}`}
      >
        {isExpanded ? (
          <ChevronDown className="w-4 h-4 flex-shrink-0 transition-transform duration-200" />
        ) : (
          <ChevronRight className="w-4 h-4 flex-shrink-0 transition-transform duration-200" />
        )}
        
        {Icon && <Icon className="w-4 h-4 text-gray-600 flex-shrink-0" />}
        
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium truncate">{title}</span>
            {badge && (
              <span className="text-xs bg-muted text-gray-600 px-2 py-0.5 rounded flex-shrink-0">
                {badge}
              </span>
            )}
          </div>
          {subtitle && (
            <div className="text-xs text-muted-foreground mt-0.5 truncate">
              {subtitle}
            </div>
          )}
        </div>
      </button>
      
      {/* Fixed animation - keep element in DOM */}
      <div
        className={`grid transition-all duration-300 ease-in-out ${
          isExpanded ? 'grid-rows-[1fr] opacity-100' : 'grid-rows-[0fr] opacity-0'
        }`}
      >
        <div className="overflow-hidden">
          <div className={`p-4 pt-0 ${contentClassName}`}>
            {children}
          </div>
        </div>
      </div>
    </div>
  );
};

export default ExpandableSection;