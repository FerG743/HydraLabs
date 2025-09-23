import React from 'react';

const SectionTabs = ({ activeSection, onSectionChange }) => {
  const tabs = [
    { id: 'files', label: 'Files' },
    { id: 'suites', label: 'Test Suites' }
  ];

  return (
    <div className="flex mt-3 bg-gray-100 rounded p-1">
      {tabs.map(tab => (
        <button
          key={tab.id}
          className={`flex-1 px-3 py-1 text-xs font-medium rounded transition-all duration-300 ${
            activeSection === tab.id 
              ? 'bg-white text-gray-900 shadow-sm animate-scale-in' 
              : 'text-gray-500 hover:text-gray-700 hover:bg-gray-50'
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