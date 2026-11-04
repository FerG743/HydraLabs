import React from 'react';
import { Search } from 'lucide-react';

const SearchBar = ({ searchQuery, onSearchChange, placeholder = "Search tests..." }) => {
  return (
    <div className="relative">
      <Search className="absolute left-2 top-1/2 transform -translate-y-1/2 w-3 h-3 text-muted-foreground" />
      <input
        type="text"
        placeholder={placeholder}
        value={searchQuery}
        onChange={(e) => onSearchChange(e.target.value)}
        className="w-full pl-7 pr-3 py-1 text-sm border border-border rounded focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all duration-300 focus:scale-[1.02]"
      />
    </div>
  );
};

export default SearchBar;