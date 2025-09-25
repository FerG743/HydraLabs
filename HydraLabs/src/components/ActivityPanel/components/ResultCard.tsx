import React from 'react';
import { useResultCard } from '../hooks';

// ResultCard Component (45 lines)
const ResultCard = ({
  result,
  index = 0,
  onClick,
  className = '',
  animationDelay = 100,
  showTimestamp = true,
  showAllFields = true,
  maxFields = 4,
  layout = 'grid' // 'grid', 'list', 'compact'
}) => {
  const {
    StatusIcon,
    statusColor,
    title,
    formattedTimestamp,
    displayFields,
    handleClick
  } = useResultCard(result, index, onClick, { showAllFields, maxFields });

  const layoutClasses = {
    grid: 'grid grid-cols-2 md:grid-cols-4 gap-4 text-sm',
    list: 'space-y-2 text-sm',
    compact: 'flex flex-wrap gap-4 text-sm'
  };

  return (
    <div
      className={`p-4 border border-gray-200 rounded-lg hover:shadow-sm transition-all 
        duration-200 cursor-pointer animate-fade-in-up hover:border-gray-300 
        hover:scale-[1.01] ${className} ${
        onClick ? 'cursor-pointer' : 'cursor-default'
      }`}
      style={{ animationDelay: `${index * animationDelay}ms` }}
      onClick={handleClick}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-3 min-w-0">
          <StatusIcon className={`w-4 h-4 ${statusColor} flex-shrink-0`} />
          <h4 className="font-medium text-gray-900 truncate">{title}</h4>
        </div>
        {showTimestamp && formattedTimestamp && (
          <span className="text-xs text-gray-500 flex-shrink-0 ml-2">
            {formattedTimestamp}
          </span>
        )}
      </div>

      {/* Fields */}
      {displayFields.length > 0 && (
        <div className={layoutClasses[layout]}>
          {displayFields.map(([key, value]) => (
            <div key={key} className="min-w-0">
              <span className="text-gray-500 capitalize block">
                {key.replace(/([A-Z])/g, ' $1').toLowerCase()}:
              </span>
              <span className="font-medium text-gray-900 block truncate" title={String(value)}>
                {String(value)}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Progress indicator for test results */}
      {result.tests && (result.passed || result.failed) && (
        <div className="mt-3 w-full bg-gray-200 rounded-full h-1.5">
          <div
            className={`h-1.5 rounded-full transition-all duration-500 ${
              result.status === 'passed' ? 'bg-green-500' :
              result.status === 'failed' ? 'bg-red-500' : 'bg-yellow-500'
            }`}
            style={{ 
              width: `${((result.passed || 0) / (result.tests || 1)) * 100}%` 
            }}
          />
        </div>
      )}
    </div>
  );
};

export default ResultCard;