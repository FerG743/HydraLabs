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