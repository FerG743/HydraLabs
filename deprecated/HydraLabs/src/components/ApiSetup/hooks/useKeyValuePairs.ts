import { useCallback } from 'react';

export const useKeyValuePairs = (data, updateData, dataPath) => {
  const addKeyValue = useCallback((type = 'basic') => {
    const newItem = type === 'formData' 
      ? { key: '', value: '', type: 'text', enabled: true }
      : { key: '', value: '', enabled: true };
    
    const currentArray = dataPath ? data[dataPath] : data;
    updateData(dataPath || 'items', [...currentArray, newItem]);
  }, [data, updateData, dataPath]);

  const removeKeyValue = useCallback((index) => {
    const currentArray = dataPath ? data[dataPath] : data;
    const newArray = currentArray.filter((_, i) => i !== index);
    updateData(dataPath || 'items', newArray);
  }, [data, updateData, dataPath]);

  const updateKeyValue = useCallback((index, field, value) => {
    const currentArray = dataPath ? data[dataPath] : data;
    const newArray = currentArray.map((item, i) => 
      i === index ? { ...item, [field]: value } : item
    );
    updateData(dataPath || 'items', newArray);
  }, [data, updateData, dataPath]);

  const clearAll = useCallback(() => {
    updateData(dataPath || 'items', [{ key: '', value: '', enabled: true }]);
  }, [updateData, dataPath]);

  const getEnabledItems = useCallback(() => {
    const currentArray = dataPath ? data[dataPath] : data;
    return currentArray.filter(item => item.key && item.enabled);
  }, [data, dataPath]);

  return {
    addKeyValue,
    removeKeyValue,
    updateKeyValue,
    clearAll,
    getEnabledItems
  };
};