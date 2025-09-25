import { useState, useCallback } from 'react';

interface UseExpandableItemsOptions {
  initialExpanded?: string[] | string;
  multipleExpansion?: boolean; // Allow multiple items expanded at once
}

export const useExpandableItems = (
  initialExpanded: string[] | string = [],
  options: UseExpandableItemsOptions = {}
) => {
  const { multipleExpansion = true } = options;
  
  const [expandedItems, setExpandedItems] = useState<string[]>(() => {
    // Ensure we always return an array, handle both array and single value
    if (Array.isArray(initialExpanded)) {
      return initialExpanded;
    }
    return initialExpanded ? [initialExpanded] : [];
  });

  const toggle = useCallback((itemId: string) => {
    setExpandedItems(prev => {
      if (prev.includes(itemId)) {
        // Always allow collapsing
        return prev.filter(id => id !== itemId);
      } else {
        // If multiple expansion is disabled, replace current with new item
        if (!multipleExpansion) {
          return [itemId];
        }
        // Otherwise add to existing expanded items
        return [...prev, itemId];
      }
    });
  }, [multipleExpansion]);

  const expand = useCallback((itemId: string) => {
    setExpandedItems(prev => {
      if (prev.includes(itemId)) {
        return prev; // Already expanded, no change needed
      }
      
      if (!multipleExpansion) {
        return [itemId]; // Replace all with just this item
      }
      
      return [...prev, itemId]; // Add to existing
    });
  }, [multipleExpansion]);

  const collapse = useCallback((itemId: string) => {
    setExpandedItems(prev => prev.filter(id => id !== itemId));
  }, []);

  const expandAll = useCallback((allItemIds: string[]) => {
    if (!multipleExpansion && allItemIds.length > 0) {
      // If multiple expansion is disabled, just expand the first item
      setExpandedItems([allItemIds[0]]);
    } else {
      setExpandedItems(allItemIds);
    }
  }, [multipleExpansion]);

  const collapseAll = useCallback(() => {
    setExpandedItems([]);
  }, []);

  const isExpanded = useCallback((itemId: string) => {
    return expandedItems.includes(itemId);
  }, [expandedItems]);

  // Helper to get count of expanded items
  const expandedCount = expandedItems.length;

  // Helper to check if any items are expanded
  const hasExpanded = expandedItems.length > 0;

  return {
    expandedItems,
    expandedCount,
    hasExpanded,
    toggle,
    expand,
    collapse,
    expandAll,
    collapseAll,
    isExpanded,
    setExpandedItems // Keep for advanced use cases
  };
};