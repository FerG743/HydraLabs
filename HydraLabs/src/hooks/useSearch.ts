import { useState, useMemo } from 'react';

export const useSearch = (data, searchFields = ['name'], options = {}) => {
  const {
    caseSensitive = false,
    matchMode = 'includes', // 'includes' | 'startsWith' | 'exact'
    debounceMs = 0
  } = options;

  const [searchQuery, setSearchQuery] = useState('');

  const filteredData = useMemo(() => {
    if (!searchQuery.trim()) return data;

    const query = caseSensitive ? searchQuery : searchQuery.toLowerCase();
    
    return data.filter(item => {
      return searchFields.some(field => {
        const value = getNestedValue(item, field);
        if (!value) return false;
        
        const searchValue = caseSensitive ? value : value.toLowerCase();
        
        switch (matchMode) {
          case 'startsWith':
            return searchValue.startsWith(query);
          case 'exact':
            return searchValue === query;
          case 'includes':
          default:
            return searchValue.includes(query);
        }
      });
    });
  }, [data, searchQuery, searchFields, caseSensitive, matchMode]);

  return {
    searchQuery,
    setSearchQuery,
    filteredData,
    hasResults: filteredData.length > 0,
    resultCount: filteredData.length
  };
};

// Helper function for nested object access
const getNestedValue = (obj, path) => {
  return path.split('.').reduce((current, key) => current?.[key], obj);
};