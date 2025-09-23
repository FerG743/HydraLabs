import React from 'react';
import { Play, RotateCcw, Search } from 'lucide-react';
import SearchBar from './SearchBar';
import SectionTabs from './SectionTabs';

const Header = ({ 
  searchQuery, 
  onSearchChange, 
  activeSection, 
  onSectionChange 
}) => {
  return (
    <div className="p-3 border-b border-gray-200">
      <div className="flex items-center justify-between mb-3">
        <h2 className="font-semibold text-sm">Side Explorer</h2>
        <div className="flex items-center gap-1">
          <button 
            className="p-1 hover:bg-gray-100 rounded text-green-600 transition-all duration-200 hover:scale-110 active:scale-95" 
            title="Run All Tests"
          >
            <Play className="w-4 h-4" />
          </button>
          <button 
            className="p-1 hover:bg-gray-100 rounded transition-all duration-200 hover:scale-110 active:scale-95 hover:animate-spin-slow" 
            title="Refresh"
          >
            <RotateCcw className="w-4 h-4" />
          </button>
          <button 
            className="p-1 hover:bg-gray-100 rounded transition-all duration-200 hover:scale-110 active:scale-95" 
            title="Search"
          >
            <Search className="w-4 h-4" />
          </button>
        </div>
      </div>
      
      <SearchBar 
        searchQuery={searchQuery}
        onSearchChange={onSearchChange}
        placeholder="Search tests..."
      />

      <SectionTabs 
        activeSection={activeSection}
        onSectionChange={onSectionChange}
      />
    </div>
  );
};

export default Header;