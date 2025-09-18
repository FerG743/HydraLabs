/ hooks/useResizer.js - HIGHLY REUSABLE ✨
// Can be used for any resizable panel, sidebar, modal, etc.
import { useState, useEffect, useCallback } from 'react';

export const useResizer = ({ 
  initialWidth = 300, 
  minWidth = 200, 
  maxWidth = 500, 
  onWidthChange,
  direction = 'horizontal' // 'horizontal' | 'vertical'
}) => {
  const [size, setSize] = useState(initialWidth);
  const [isResizing, setIsResizing] = useState(false);

  const handleMouseDown = useCallback((e) => {
    setIsResizing(true);
    e.preventDefault();
  }, []);

  const handleMouseMove = useCallback((e) => {
    if (!isResizing) return;
    
    const newSize = direction === 'horizontal' ? e.clientX : e.clientY;
    const constrainedSize = Math.min(Math.max(newSize, minWidth), maxWidth);
    
    setSize(constrainedSize);
    onWidthChange?.(constrainedSize);
  }, [isResizing, minWidth, maxWidth, onWidthChange, direction]);

  const handleMouseUp = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = direction === 'horizontal' ? 'col-resize' : 'row-resize';
      document.body.style.userSelect = 'none';
    } else {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizing, handleMouseMove, handleMouseUp, direction]);

  return {
    size,
    isResizing,
    handleMouseDown,
    setSize // Allow manual size changes
  };
};