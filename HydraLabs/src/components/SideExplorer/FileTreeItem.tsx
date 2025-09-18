import React from 'react';
import { ChevronRight, ChevronDown, Folder, FileText } from 'lucide-react';

const FileTreeItem = ({ 
  item, 
  level = 0, 
  expandedFolders, 
  onToggleFolder, 
  onFileSelect 
}) => {
  const isExpanded = expandedFolders.includes(item.name);
  
  return (
    <div>
      <div 
        className="flex items-center gap-1 px-2 py-1 hover:bg-gray-100 cursor-pointer text-sm group transition-all duration-200 hover:scale-[1.01] animate-fade-in"
        style={{ paddingLeft: `${8 + level * 16}px` }}
        onClick={() => {
          if (item.type === 'folder') {
            onToggleFolder(item.name);
          } else if (onFileSelect) {
            onFileSelect(item);
          }
        }}
      >
        {item.type === 'folder' ? (
          <>
            {isExpanded ? 
              <ChevronDown className="w-3 h-3 text-gray-500" /> : 
              <ChevronRight className="w-3 h-3 text-gray-500" />
            }
            <Folder className="w-4 h-4 text-blue-500" />
          </>
        ) : (
          <>
            <div className="w-3 h-3" />
            <FileText className="w-4 h-4 text-gray-500" />
          </>
        )}
        <span className="truncate flex-1">{item.name}</span>
        {item.size && (
          <span className="text-xs text-gray-400 opacity-0 group-hover:opacity-100">
            {item.size}
          </span>
        )}
      </div>
      
      {item.type === 'folder' && isExpanded && item.children && (
        <div className="animate-slide-down">
          {item.children.map((child, idx) => (
            <div 
              key={idx} 
              className="animate-fade-in-up" 
              style={{ animationDelay: `${idx * 50}ms` }}
            >
              <FileTreeItem 
                item={child} 
                level={level + 1}
                expandedFolders={expandedFolders}
                onToggleFolder={onToggleFolder}
                onFileSelect={onFileSelect}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default FileTreeItem;