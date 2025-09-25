import React from 'react';
import { useResizer } from '@/hooks'; // Still fix this typo if needed
import { useExplorerState } from './hooks'; 
import { defaultTestSuites, defaultFileTree } from './Utils';
import { 
  ResizeHandle,    // This will import from resizeHandle.tsx
  Header,          // This will import from header.tsx
  FileTreeSection,
  TestSuiteSection,
  FooterStats 
} from './components';
export const SideExplorer = ({ 
  width: initialWidth = 300,
  minWidth = 200,
  maxWidth = 500,
  onTestSuiteSelect,
  onFileSelect,
  onWidthChange,
  activeTestSuite,
  testSuites = [],
  fileTree = []
} = {} ) => {
  // Get data with defaults
  const currentTestSuites = testSuites.length > 0 ? testSuites : defaultTestSuites;
  const currentFileTree = fileTree.length > 0 ? fileTree : defaultFileTree;

  // Custom hooks
  const { width, isResizing, handleMouseDown } = useResizer({
    initialWidth,
    minWidth,
    maxWidth,
    onWidthChange
  });

  const {
    expandedFolders,
    searchQuery,
    setSearchQuery,
    activeSection,
    setActiveSection,
    toggleFolder,
    filteredTestSuites
  } = useExplorerState(currentTestSuites, currentFileTree);

  return (
    <div className="flex h-full">
      <div 
        className="bg-white border-r border-gray-200 flex flex-col h-full"
        style={{ width: `${width}px` }}
      >
        {/* Header */}
        <Header
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          activeSection={activeSection}
          onSectionChange={setActiveSection}
        />
        
        {/* Content */}
        <div className="flex-1 overflow-y-auto">
          {activeSection === 'files' ? (
            <FileTreeSection
              fileTree={currentFileTree}
              expandedFolders={expandedFolders}
              onToggleFolder={toggleFolder}
              onFileSelect={onFileSelect}
            />
          ) : (
            <TestSuitesSection
              testSuites={filteredTestSuites}
              activeTestSuite={activeTestSuite}
              onTestSuiteSelect={onTestSuiteSelect}
            />
          )}
        </div>

        {/* Footer Stats */}
        <FooterStats
          activeSection={activeSection}
          testSuites={currentTestSuites}
          fileTree={currentFileTree}
        />
      </div>
      
      {/* Resize Handle */}
      <ResizeHandle
        isResizing={isResizing}
        onMouseDown={handleMouseDown}
      />
    </div>
  );
};

export default SideExplorer;