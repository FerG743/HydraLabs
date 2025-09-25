import React from "react";
import ReactDOM from "react-dom/client";
import APITesting from "./pages/APITesting";
import "./App.css";

// Apply dark theme class to enable your CSS variables
document.documentElement.classList.add('dark');

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <APITesting />
  </React.StrictMode>
);