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
        // FIXED: Changed from light theme to your dark theme
        className="flex items-center gap-1 px-2 py-1 hover:bg-muted cursor-pointer text-sm group transition-all duration-200 hover:scale-[1.01] animate-fade-in text-foreground"
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
              <ChevronDown className="w-3 h-3 text-muted-foreground" /> : 
              <ChevronRight className="w-3 h-3 text-muted-foreground" />
            }
            {/* FIXED: Changed folder color to electric lime instead of blue */}
            <Folder className="w-4 h-4 text-primary" />
          </>
        ) : (
          <>
            <div className="w-3 h-3" />
            <FileText className="w-4 h-4 text-muted-foreground" />
          </>
        )}
        <span className="truncate flex-1 text-foreground">{item.name}</span>
        {item.size && (
          <span className="text-xs text-muted-foreground opacity-0 group-hover:opacity-100">
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