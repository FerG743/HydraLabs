import React from 'react';

const SectionTabs = ({ activeSection, onSectionChange }) => {
  const tabs = [
    { id: 'files', label: 'Files' },
    { id: 'suites', label: 'Test Suites' }
  ];

  return (
    <div className="flex mt-3 bg-muted rounded p-1">
      {tabs.map(tab => (
        <button
          key={tab.id}
          className={`flex-1 px-3 py-1 text-xs font-medium rounded transition-all duration-300 ${
            activeSection === tab.id 
              ? 'bg-card text-foreground shadow-sm animate-scale-in' 
              : 'text-muted-foreground hover:text-foreground hover:bg-muted'
          }`}
          onClick={() => onSectionChange(tab.id)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
};

export default SectionTabs;