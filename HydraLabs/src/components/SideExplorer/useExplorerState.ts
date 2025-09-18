import { useState } from 'react';

export const useExplorerState = (initialTestSuites = [], initialFileTree = []) => {
  const [expandedFolders, setExpandedFolders] = useState(['test-suites', 'api-tests']);
  const [searchQuery, setSearchQuery] = useState('');
  const [activeSection, setActiveSection] = useState('files');

  const toggleFolder = (folderName) => {
    setExpandedFolders(prev => 
      prev.includes(folderName) 
        ? prev.filter(f => f !== folderName)
        : [...prev, folderName]
    );
  };

  const filteredTestSuites = initialTestSuites.filter(suite => 
    suite.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    suite.type.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return {
    expandedFolders,
    searchQuery,
    setSearchQuery,
    activeSection,
    setActiveSection,
    toggleFolder,
    filteredTestSuites
  };
};