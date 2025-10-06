import React from 'react';
import { ChevronRight, ChevronDown, Folder, FileText } from 'lucide-react';
import { AnimatedCollapse } from '../../Animations/index'

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
        className="flex items-center gap-1 px-2 py-1 hover:bg-muted cursor-pointer text-sm group transition-all duration-200 hover:scale-[1.01] text-foreground rounded"
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
            <Folder className="w-4 h-4 text-primary transition-all group-hover:drop-shadow-[0_0_4px_hsl(var(--primary))]" />
          </>
        ) : (
          <>
            <div className="w-3 h-3" />
            <FileText className="w-4 h-4 text-muted-foreground group-hover:text-foreground transition-colors" />
          </>
        )}
        <span className="truncate flex-1 text-foreground group-hover:text-primary transition-colors">{item.name}</span>
        {item.size && (
          <span className="text-xs text-muted-foreground opacity-0 group-hover:opacity-100 transition-opacity">
            {item.size}
          </span>
        )}
      </div>
      
      {item.type === 'folder' && item.children && (
        <AnimatedCollapse isOpen={isExpanded} staggerChildren staggerDelay={30}>
          {item.children.map((child, idx) => (
            <FileTreeItem 
              key={idx}
              item={child} 
              level={level + 1}
              expandedFolders={expandedFolders}
              onToggleFolder={onToggleFolder}
              onFileSelect={onFileSelect}
            />
          ))}
        </AnimatedCollapse>
      )}
    </div>
  );
};

export default FileTreeItem;