import React from 'react';
import { XCircle } from 'lucide-react';
import { getStatusIcon, getTypeColor } from '../Utils'; 


const TestSuiteCard = ({ 
  suite, 
  isActive, 
  index, 
  onSelect 
}) => {
  return (
    <div 
      className={`p-3 rounded border cursor-pointer transition-all duration-300 hover:shadow-sm group animate-fade-in-up ${
        isActive 
          ? 'border-blue-500 bg-blue-50 shadow-sm scale-[1.02] animate-scale-in' 
          : 'border-border hover:border-gray-300 hover:bg-muted hover:scale-[1.01]'
      }`}
      style={{ animationDelay: `${index * 100}ms` }}
      onClick={() => onSelect && onSelect(suite)}
    >
      <div className="flex items-start justify-between mb-2">
        <div className="flex items-center gap-2 min-w-0">
          <div className={`transition-all duration-300 ${suite.status === 'running' ? 'animate-pulse-subtle' : ''}`}>
            {getStatusIcon(suite.status)}
          </div>
          <span className="text-sm font-medium truncate">{suite.name}</span>
        </div>
        {suite.status === 'running' && (
          <button className="p-1 hover:bg-border rounded opacity-0 group-hover:opacity-100 transition-all duration-200 hover:scale-110">
            <XCircle className="w-3 h-3 text-muted-foreground" />
          </button>
        )}
      </div>
      
      <div className="flex items-center gap-2 mb-2">
        <span className={`px-2 py-0.5 text-xs font-medium rounded transition-all duration-200 animate-fade-in ${getTypeColor(suite.type)}`}>
          {suite.type}
        </span>
      </div>
      
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>{suite.passed}/{suite.tests} passed</span>
        <span>{suite.lastRun}</span>
      </div>
      
      {suite.failed > 0 && (
        <div className="mt-2 text-xs text-red-600 animate-fade-in">
          {suite.failed} failed test{suite.failed !== 1 ? 's' : ''}
        </div>
      )}
      
      {/* Progress Bar */}
      <div className="mt-2 w-full bg-border rounded-full h-1 overflow-hidden">
        <div 
          className={`h-1 rounded-full transition-all duration-700 ease-out ${
            suite.status === 'failed' ? 'bg-red-500' : 
            suite.status === 'running' ? 'bg-yellow-500 animate-shimmer' : 'bg-green-500'
          } ${suite.status === 'running' ? 'animate-pulse-subtle' : ''}`}
          style={{ 
            width: `${(suite.passed / suite.tests) * 100}%`,
            animationDelay: `${index * 150}ms`
          }}
        />
      </div>
    </div>
  );
};

export default TestSuiteCard;