// hooks/useExpandableItems.js - REUSABLE ✨

import { useState, useCallback } from 'react';

export const useExpandableItems = (initialExpanded = []) => {
  const [expandedItems, setExpandedItems] = useState(
    Array.isArray(initialExpanded) ? initialExpanded : [initialExpanded]
  );

  const toggle = useCallback((itemId) => {
    setExpandedItems(prev => 
      prev.includes(itemId) 
        ? prev.filter(id => id !== itemId)
        : [...prev, itemId]
    );
  }, []);

  const expand = useCallback((itemId) => {
    setExpandedItems(prev => 
      prev.includes(itemId) ? prev : [...prev, itemId]
    );
  }, []);

  const collapse = useCallback((itemId) => {
    setExpandedItems(prev => prev.filter(id => id !== itemId));
  }, []);

  const expandAll = useCallback((allItemIds) => {
    setExpandedItems(allItemIds);
  }, []);

  const collapseAll = useCallback(() => {
    setExpandedItems([]);
  }, []);

  const isExpanded = useCallback((itemId) => {
    return expandedItems.includes(itemId);
  }, [expandedItems]);

  return {
    expandedItems,
    toggle,
    expand,
    collapse,
    expandAll,
    collapseAll,
    isExpanded,
    setExpandedItems
  };
};