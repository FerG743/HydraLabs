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
    // FIXED: Changed border color to use theme variable
    <div className="p-3 border-b border-border">
      <div className="flex items-center justify-between mb-3">
        {/* FIXED: Added proper text color */}
        <h2 className="font-semibold text-sm text-foreground">Side Explorer</h2>
        <div className="flex items-center gap-1">
          <button 
            // FIXED: Changed hover and icon colors to use theme
            className="p-1 hover:bg-muted rounded text-success transition-all duration-200 hover:scale-110 active:scale-95" 
            title="Run All Tests"
          >
            <Play className="w-4 h-4" />
          </button>
          <button 
            // FIXED: Changed hover background and text color
            className="p-1 hover:bg-muted rounded text-muted-foreground transition-all duration-200 hover:scale-110 active:scale-95 hover:animate-spin-slow" 
            title="Refresh"
          >
            <RotateCcw className="w-4 h-4" />
          </button>
          <button 
            // FIXED: Changed hover background and text color
            className="p-1 hover:bg-muted rounded text-muted-foreground transition-all duration-200 hover:scale-110 active:scale-95" 
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