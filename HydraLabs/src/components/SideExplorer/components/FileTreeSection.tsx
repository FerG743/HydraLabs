import React from 'react';
import { Plus } from 'lucide-react';
import FileTreeItem from './FileTreeItem';

const FileTreeSection = ({ 
  fileTree, 
  expandedFolders, 
  onToggleFolder, 
  onFileSelect 
}) => {
  return (
    <div className="p-2">
      <div className="flex items-center justify-between mb-2">
        {/* FIXED: Changed from gray text to muted foreground */}
        <span className="text-xs font-medium text-muted-foreground px-2">PROJECT FILES</span>
        <button 
          // FIXED: Changed hover from gray to muted, and icon color
          className="p-1 hover:bg-muted rounded transition-all duration-200 hover:scale-110" 
          title="New File"
        >
          <Plus className="w-3 h-3 text-muted-foreground" />
        </button>
      </div>
      {fileTree.map((item, idx) => (
        <FileTreeItem 
          key={idx} 
          item={item}
          expandedFolders={expandedFolders}
          onToggleFolder={onToggleFolder}
          onFileSelect={onFileSelect}
        />
      ))}
    </div>
  );
};

export default FileTreeSection;