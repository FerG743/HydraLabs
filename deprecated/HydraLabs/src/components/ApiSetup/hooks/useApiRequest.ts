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
    },
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