"use client"

import React, { useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, BarChart, Bar } from 'recharts';

const usageData = [
  { time: '10:00', tokens: 4000, cost: 0.05 },
  { time: '10:05', tokens: 3000, cost: 0.03 },
  { time: '10:10', tokens: 2000, cost: 0.02 },
  { time: '10:15', tokens: 2780, cost: 0.04 },
  { time: '10:20', tokens: 1890, cost: 0.01 },
  { time: '10:25', tokens: 2390, cost: 0.02 },
  { time: '10:30', tokens: 3490, cost: 0.04 },
];

const tenants = [
  { id: '1', name: 'Acme Corp', tier: 'Pro', cost: '$12.40' },
  { id: '2', name: 'Startup Inc', tier: 'Basic', cost: '$3.20' },
  { id: '3', name: 'Global Tech', tier: 'Enterprise', cost: '$145.00' },
];

export default function Dashboard() {
  return (
    <div className="min-h-screen bg-gray-50 p-8 font-sans">
      <header className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">GateKeeper Admin</h1>
        <p className="text-gray-500">LLM API Gateway & Intelligent Rate Limiter</p>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-100">
          <h3 className="text-sm font-medium text-gray-500">Total Tokens (24h)</h3>
          <p className="text-3xl font-bold text-gray-900">1.2M</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-100">
          <h3 className="text-sm font-medium text-gray-500">Total Cost (24h)</h3>
          <p className="text-3xl font-bold text-gray-900">$18.45</p>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-100">
          <h3 className="text-sm font-medium text-gray-500">Cache Hit Rate</h3>
          <p className="text-3xl font-bold text-emerald-600">34.2%</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-100">
          <h2 className="text-xl font-semibold mb-4">Token Usage</h2>
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
          <h2 className="text-xl font-semibold mb-4">Cost Over Time (USD)</h2>
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

      <div className="bg-white rounded-lg shadow-sm border border-gray-100 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100">
          <h2 className="text-xl font-semibold">Tenants & API Keys</h2>
        </div>
        <table className="w-full text-left">
          <thead className="bg-gray-50 text-gray-500 text-sm">
            <tr>
              <th className="px-6 py-3 font-medium">Tenant Name</th>
              <th className="px-6 py-3 font-medium">Tier</th>
              <th className="px-6 py-3 font-medium">Current Spend</th>
              <th className="px-6 py-3 font-medium text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {tenants.map((tenant) => (
              <tr key={tenant.id} className="hover:bg-gray-50">
                <td className="px-6 py-4 font-medium text-gray-900">{tenant.name}</td>
                <td className="px-6 py-4 text-gray-500">
                  <span className="px-2 py-1 text-xs rounded-full bg-blue-50 text-blue-600 font-medium">
                    {tenant.tier}
                  </span>
                </td>
                <td className="px-6 py-4 text-gray-500">{tenant.cost}</td>
                <td className="px-6 py-4 text-right">
                  <button className="text-indigo-600 hover:text-indigo-900 text-sm font-medium">
                    Manage Policies
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
