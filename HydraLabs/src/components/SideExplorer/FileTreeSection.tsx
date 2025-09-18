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
        <span className="text-xs font-medium text-gray-500 px-2">PROJECT FILES</span>
        <button 
          className="p-1 hover:bg-gray-100 rounded transition-all duration-200 hover:scale-110" 
          title="New File"
        >
          <Plus className="w-3 h-3 text-gray-400" />
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