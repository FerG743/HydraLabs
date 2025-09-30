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

  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    console.log('Mouse down triggered', direction);
    setIsResizing(true);
    startPos.current = direction === 'horizontal' ? e.clientX : e.clientY;
    startSize.current = size;
    e.preventDefault();
  }, [direction, size]);

  const handleMouseMove = useCallback((e: MouseEvent) => {
    if (!isResizing) return;
    
    const currentPos = direction === 'horizontal' ? e.clientX : e.clientY;
    const delta = currentPos - startPos.current;
    
    console.log('Moving:', { currentPos, startPos: startPos.current, delta });
    
    // Invert delta for vertical direction so dragging down increases size
    const adjustedDelta = direction === 'vertical' ? -delta : delta;
    const newSize = startSize.current + adjustedDelta;
    
    const constrainedSize = Math.min(Math.max(newSize, minWidth), maxWidth);
    
    console.log('New size:', constrainedSize);
    
    setSize(constrainedSize);
    onWidthChange?.(constrainedSize);
  }, [isResizing, minWidth, maxWidth, onWidthChange, direction]);

  const handleMouseUp = useCallback(() => {
    console.log('Mouse up');
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      console.log('Attaching listeners');
      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = direction === 'horizontal' ? 'col-resize' : 'row-resize';
      document.body.style.userSelect = 'none';
    }

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    };
  }, [isResizing, handleMouseMove, handleMouseUp, direction]);

  return {
    width: size,
    isResizing,
    handleMouseDown,
    setSize
  };
};