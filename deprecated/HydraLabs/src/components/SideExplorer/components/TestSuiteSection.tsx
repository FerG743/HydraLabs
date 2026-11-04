import React from 'react';
import { Plus } from 'lucide-react';
import TestSuiteCard from './TestSuiteCard';

const TestSuitesSection = ({ 
  testSuites, 
  activeTestSuite, 
  onTestSuiteSelect 
}) => {
  return (
    <div className="p-2">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs font-medium text-muted-foreground px-2">
          TEST SUITES ({testSuites.length})
        </span>
        <button 
          className="p-1 hover:bg-muted rounded transition-all duration-200 hover:scale-110" 
          title="New Test Suite"
        >
          <Plus className="w-3 h-3 text-muted-foreground" />
        </button>
      </div>
      
      <div className="space-y-2">
        {testSuites.map((suite, index) => (
          <TestSuiteCard
            key={suite.id}
            suite={suite}
            isActive={activeTestSuite === suite.id}
            index={index}
            onSelect={onTestSuiteSelect}
          />
        ))}
      </div>
    </div>
  );
};

export default TestSuitesSection;