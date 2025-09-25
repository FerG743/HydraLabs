import { useState, useEffect, useCallback, useRef } from 'react';

interface UseResizerProps {
  initialWidth?: number;
  minWidth?: number;
  maxWidth?: number;
  onWidthChange?: (width: number) => void;
  direction?: 'horizontal' | 'vertical';
}

export const useResizer = ({
  initialWidth = 300,
  minWidth = 200,
  maxWidth = 500,
  onWidthChange,
  direction = 'horizontal'
}: UseResizerProps) => {
  const [size, setSize] = useState(initialWidth);
  const [isResizing, setIsResizing] = useState(false);
  const startPos = useRef(0);
  const startSize = useRef(0);

  const handleMouseDown = useCallback((e: MouseEvent) => {
    setIsResizing(true);
    // Store the starting position and current size
    startPos.current = direction === 'horizontal' ? e.clientX : e.clientY;
    startSize.current = size;
    e.preventDefault();
  }, [direction, size]);

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!isResizing) return;
    
    const currentPos = direction === 'horizontal' ? e.clientX : e.clientY;
    const delta = currentPos - startPos.current;
    const newSize = startSize.current + delta;
    
    // Apply constraints
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