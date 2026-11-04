// components/ResizeHandle.tsx
import React from 'react';

interface ResizeHandleProps {
  isResizing: boolean;
  onMouseDown: (e: React.MouseEvent) => void;
  direction?: 'horizontal' | 'vertical';
}

export const ResizeHandle = ({ 
  isResizing, 
  onMouseDown, 
  direction = 'horizontal' 
}: ResizeHandleProps) => {
  const isVertical = direction === 'vertical';
  
  return (
    <div
      onMouseDown={onMouseDown}
      className={`${
        isVertical 
          ? 'h-1 cursor-row-resize' 
          : 'w-1 cursor-col-resize'
      } bg-border hover:bg-primary/50 transition-colors relative group ${
        isResizing ? 'bg-primary' : ''
      }`}
    >
      {/* Wider hit area for easier grabbing */}
      <div className={
        isVertical 
          ? 'absolute inset-x-0 -top-1 -bottom-1 h-3' 
          : 'absolute inset-y-0 -left-1 -right-1 w-3'
      } />
    </div>
  );
};

export default ResizeHandle;
