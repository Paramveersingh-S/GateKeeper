"use client"

import React, { useState, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, BarChart, Bar } from 'recharts';

export default function Dashboard() {
  const [token, setToken] = useState<string | null>(null);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const [usageData, setUsageData] = useState<any[]>([]);

  useEffect(() => {
    // Check local storage on mount
    const storedToken = localStorage.getItem('gk_admin_token');
    if (storedToken) setToken(storedToken);
  }, []);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const res = await fetch('/api/admin/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });
      if (!res.ok) throw new Error('Invalid credentials');
      const data = await res.json();
      setToken(data.token);
      localStorage.setItem('gk_admin_token', data.token);
    } catch (err: any) {
      setError(err.message);
    }
  };

  const handleLogout = () => {
    setToken(null);
    localStorage.removeItem('gk_admin_token');
  };

  // Poll data
  useEffect(() => {
    if (!token) return;

    const fetchData = async () => {
      try {
        const headers = { Authorization: `Bearer ${token}` };
        // In a real app we'd fetch actual tenants. For now we use the mock tenant to see data.
        const res = await fetch('/api/admin/metrics?tenant_id=00000000-0000-0000-0000-000000000000', { headers });
        if (res.status === 401) { handleLogout(); return; }
        if (!res.ok) return;
        const data = await res.json();
        
        // Map data to recharts format
        if (data && data.length > 0) {
            const formatted = data.map((d: any) => ({
                time: new Date(d.hour).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'}),
                tokens: d.tokens,
                cost: d.cost
            }));
            setUsageData(formatted);
        }
      } catch (err) {
        console.error(err);
      }
    };

    fetchData();
    const interval = setInterval(fetchData, 5000);
    return () => clearInterval(interval);
  }, [token]);

  if (!token) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center font-sans">
        <form onSubmit={handleLogin} className="bg-white p-8 rounded-lg shadow-sm border border-gray-100 w-96">
          <h1 className="text-2xl font-bold mb-6 text-gray-900 text-center">GateKeeper Admin</h1>
          {error && <div className="mb-4 text-red-500 text-sm bg-red-50 p-3 rounded">{error}</div>}
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
            <input type="text" value={username} onChange={e => setUsername(e.target.value)} className="w-full border-gray-300 rounded-md shadow-sm p-2 text-gray-900 border" required />
          </div>
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
            <input type="password" value={password} onChange={e => setPassword(e.target.value)} className="w-full border-gray-300 rounded-md shadow-sm p-2 text-gray-900 border" required />
          </div>
          <button type="submit" className="w-full bg-indigo-600 text-white font-medium py-2 px-4 rounded-md hover:bg-indigo-700 transition">Login (admin / admin)</button>
        </form>
      </div>
    );
  }

  const totalTokens = usageData.reduce((acc, curr) => acc + curr.tokens, 0);
  const totalCost = usageData.reduce((acc, curr) => acc + curr.cost, 0);

  return (
    <div className="min-h-screen bg-gray-50 p-8 font-sans text-gray-900">
      <header className="mb-8 flex justify-between items-center">
        <div>
            <h1 className="text-3xl font-bold text-gray-900">GateKeeper Admin</h1>
            <p className="text-gray-500">LLM API Gateway & Intelligent Rate Limiter</p>
        </div>
        <button onClick={handleLogout} className="text-sm text-indigo-600 hover:text-indigo-900 font-medium">Logout</button>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-100">
          <h3 className="text-sm font-medium text-gray-500">Total Tokens (24h)</h3>
          <p className="text-3xl font-bold text-gray-900">{totalTokens.toLocaleString()}</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-100">
          <h3 className="text-sm font-medium text-gray-500">Total Cost (24h)</h3>
          <p className="text-3xl font-bold text-gray-900">${totalCost.toFixed(5)}</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-100">
          <h3 className="text-sm font-medium text-gray-500">Cache Hit Rate</h3>
          <p className="text-3xl font-bold text-emerald-600">--%</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-100">
          <h2 className="text-xl font-semibold mb-4 text-gray-900">Token Usage</h2>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={usageData}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Line type="monotone" dataKey="tokens" stroke="#8884d8" activeDot={{ r: 8 }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-100">
          <h2 className="text-xl font-semibold mb-4 text-gray-900">Cost Over Time (USD)</h2>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={usageData}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="cost" fill="#10b981" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>
    </div>
  );
}
