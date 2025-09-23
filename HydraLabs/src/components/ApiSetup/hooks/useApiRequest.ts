// hooks/useApiRequest.js - Main API request state management
import { useState, useCallback } from 'react';

export const useApiRequest = (initialData = {}) => {
  const [requestData, setRequestData] = useState({
    method: 'GET',
    url: '',
    headers: [{ key: '', value: '', enabled: true }],
    params: [{ key: '', value: '', enabled: true }],
    body: {
      type: 'none',
      raw: '',
      formData: [{ key: '', value: '', type: 'text', enabled: true }]
    auth: {
      type: 'none',
      bearer: '',
      basic: { username: '', password: '' },
      apiKey: { key: '', value: '', addTo: 'header' }
    },
    tests: '',
    preRequest: '',
    ...initialData
  });

  const [isLoading, setIsLoading] = useState(false);

  const updateRequestData = useCallback((path, value) => {
    setRequestData(prev => {
      const newData = { ...prev };
      const keys = path.split('.');
      let current = newData;
      
      for (let i = 0; i < keys.length - 1; i++) {
        current = current[keys[i]];
      }
      
      current[keys[keys.length - 1]] = value;
      return newData;
    });
  }, []);

  const resetRequest = useCallback(() => {
    setRequestData({
      method: 'GET',
      url: '',
      headers: [{ key: '', value: '', enabled: true }],
      params: [{ key: '', value: '', enabled: true }],
      body: {
        type: 'none',
        raw: '',
        formData: [{ key: '', value: '', type: 'text', enabled: true }]
      },
      auth: {
        type: 'none',
        bearer: '',
        basic: { username: '', password: '' },
        apiKey: { key: '', value: '', addTo: 'header' }
      },
      tests: '',
      preRequest: ''
    });
  }, []);

  return {
    requestData,
    setRequestData,
    updateRequestData,
    resetRequest,
    isLoading,
    setIsLoading
  };
};

// hooks/useKeyValuePairs.js - Reusable for headers, params, form data
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

// hooks/usePasswordVisibility.js - Reusable for password fields
import { useState, useCallback } from 'react';

export const usePasswordVisibility = () => {
  const [showPasswords, setShowPasswords] = useState({});

  const togglePassword = useCallback((fieldId) => {
    setShowPasswords(prev => ({
      ...prev,
      [fieldId]: !prev[fieldId]
    }));
  }, []);

  const isVisible = useCallback((fieldId) => {
    return showPasswords[fieldId] || false;
  }, [showPasswords]);

  const hideAll = useCallback(() => {
    setShowPasswords({});
  }, []);

  return {
    togglePassword,
    isVisible,
    hideAll
  };
};