import React from 'react';
import { useMetricCard } from './hooks/useMetricCard';

// MetricCard Component (35 lines)
const MetricCard = ({
  metric,
  index = 0,
  onClick,
  className = '',
  animationDelay = 150,
  showIcon = true,
  iconColor = 'text-blue-500',
  size = 'default' // 'small', 'default', 'large'
}) => {
  const {
    key,
    value,
    formattedKey,
    formattedValue,
    MetricIcon,
    handleClick
  } = useMetricCard(metric, onClick);

  const sizeClasses = {
    small: 'p-3',
    default: 'p-4', 
    large: 'p-6'
  };

  const valueSizeClasses = {
    small: 'text-lg',
    default: 'text-2xl',
    large: 'text-3xl'
  };

  return (
    <div
      className={`bg-gray-50 rounded-lg animate-scale-in transition-all duration-200 
        hover:shadow-sm hover:bg-gray-100 ${sizeClasses[size]} ${className} ${
        onClick ? 'cursor-pointer' : ''
      }`}
      style={{ animationDelay: `${index * animationDelay}ms` }}
      onClick={handleClick}
    >
      {/* Header with icon and label */}
      <div className="flex items-center gap-2 mb-2">
        {showIcon && (
          <MetricIcon className={`w-5 h-5 ${iconColor}`} />
        )}
        <span className="text-sm text-gray-600 capitalize font-medium">
          {formattedKey}
        </span>
      </div>
      
      {/* Value */}
      <div className={`font-bold text-gray-900 ${valueSizeClasses[size]}`}>
        {formattedValue}
      </div>
    </div>
  );
};

export default MetricCard;