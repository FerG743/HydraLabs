import { useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import "./App.css";

function App() {
  const [greetMsg, setGreetMsg] = useState("");
  const [name, setName] = useState("");

  async function greet() {
    setGreetMsg(await invoke("greet", { name }));
  }

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <h1 className="text-4xl font-bold text-gray-800 mb-8 text-center">
        Welcome to Tauri + Tailwind! 🚀
      </h1>

      <div className="max-w-md mx-auto space-y-4">
        <input
          className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          onChange={(e) => setName(e.currentTarget.value)}
          placeholder="Enter a name..."
          value={name}
        />
        
        <button 
          onClick={greet}
          className="w-full px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors font-medium"
        >
          Greet
        </button>

        <button 
          onClick={() => setGreetMsg("")}
          className="w-full px-4 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300 transition-colors"
        >
          Clear
        </button>

        {greetMsg && (
          <div className="p-4 rounded-lg bg-green-50 border border-green-200">
            <p className="text-center font-medium text-green-800">{greetMsg}</p>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;