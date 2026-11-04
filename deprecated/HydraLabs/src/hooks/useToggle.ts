import { useState, useCallback } from 'react';

export const useToggle = (initialState = false) => {
  const [isOn, setIsOn] = useState(initialState);

  const toggle = useCallback(() => setIsOn(prev => !prev), []);
  const turnOn = useCallback(() => setIsOn(true), []);
  const turnOff = useCallback(() => setIsOn(false), []);

  return {
    isOn,
    toggle,
    turnOn,
    turnOff,
    setIsOn
  };
};