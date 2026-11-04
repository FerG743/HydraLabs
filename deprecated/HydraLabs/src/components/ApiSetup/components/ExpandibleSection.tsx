import React from 'react';
import { ChevronDown, ChevronRight } from 'lucide-react';
import { AnimatedCollapse } from '../../Animations/index'; // Adjust path as needed

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
    <div className={`border-b border-border ${className}`}>
      <button
        onClick={onToggle}
        className={`w-full flex items-center gap-3 p-3 text-left hover:bg-muted transition-all duration-200 ${headerClassName}`}
      >
        {isExpanded ? (
          <ChevronDown className="w-4 h-4 flex-shrink-0 transition-transform duration-200" />
        ) : (
          <ChevronRight className="w-4 h-4 flex-shrink-0 transition-transform duration-200" />
        )}
        
        {Icon && <Icon className="w-4 h-4 text-muted-foreground flex-shrink-0" />}
        
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium truncate">{title}</span>
            {badge && (
              <span className="text-xs bg-muted text-muted-foreground px-2 py-0.5 rounded flex-shrink-0">
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
      
      <AnimatedCollapse isOpen={isExpanded} duration={300}>
        <div className={`p-4 pt-0 ${contentClassName}`}>
          {children}
        </div>
      </AnimatedCollapse>
    </div>
  );
};

export default ExpandableSection;