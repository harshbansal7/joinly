/**
 * Modern dashboard page with statistics and quick actions.
 */

'use client';

import { useEffect, useState, useCallback } from 'react';
import {
  Plus,
  Bot,
  Activity,
  Users,
  TrendingUp,
  Play,
  Square,
  Settings,
  CheckCircle,
  AlertCircle,
  Loader2
} from 'lucide-react';

import { Layout } from '@/components/layout/layout';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { useAgentStore, useUIStore } from '@/lib/store';
import { agentsApi, Agent, CreateAgentRequest } from '@/lib/api';
import CreateAgentPage from './agents/create/page';

export default function DashboardPage() {
  const {
    agents,
    isLoading,
    setAgents,
    updateAgent,
    setLoading,
    connectSessionWebSocket,
    disconnectSessionWebSocket
  } = useAgentStore();

  const { addNotification } = useUIStore();
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [stats, setStats] = useState({
    totalAgents: 0,
    runningAgents: 0,
    stoppedAgents: 0,
    errorAgents: 0,
    totalMeetings: 0,
    avgUptime: 0
  });

  const loadAgents = useCallback(async () => {
    setLoading(true);
    try {
      const response = await agentsApi.list();
      setAgents(response.data);

      // Calculate stats
      const totalAgents = response.data.length;
      const runningAgents = response.data.filter((a: Agent) => a.status === 'running').length;
      const stoppedAgents = response.data.filter((a: Agent) => a.status === 'stopped').length;
      const errorAgents = response.data.filter((a: Agent) => a.status === 'error').length;

      setStats({
        totalAgents,
        runningAgents,
        stoppedAgents,
        errorAgents,
        totalMeetings: 0, // TODO: Get from meetings API
        avgUptime: 85.2 // TODO: Calculate from agent data
      });
    } catch (error) {
      console.error('Failed to load agents:', error);
      addNotification({
        type: 'error',
        title: 'Failed to load dashboard',
        message: 'Could not load agents data. Please refresh the page.',
      });
    } finally {
      setLoading(false);
    }
  }, [setAgents, addNotification, setLoading]);

  useEffect(() => {
    loadAgents();
  }, [loadAgents]);

  // Real-time updates via session WebSocket
  useEffect(() => {
    connectSessionWebSocket((message) => {
      console.log('Dashboard session WebSocket message:', message);

      if (message.type === 'status') {
        const agentToUpdate = agents.find(a => a.id === message.agent_id);
        if (agentToUpdate) {
          updateAgent({ 
            ...agentToUpdate, 
            status: message.data.status as Agent['status'],
            error_message: message.data.error ? message.data.error as string : agentToUpdate.error_message
          });
        }
      } else if (message.type === 'log') {
        const agentToUpdate = agents.find(a => a.id === message.agent_id);
        if (agentToUpdate) {
          const newLog = {
            timestamp: (message.data.timestamp as string) || new Date().toISOString(),
            level: (message.data.level as string) || 'info',
            message: (message.data.message as string) || 'Unknown log message'
          };
          updateAgent({
            ...agentToUpdate,
            logs: [...(agentToUpdate.logs || []), newLog]
          });
        }
      }
    });

    return () => {
      disconnectSessionWebSocket();
    };
  }, [connectSessionWebSocket, disconnectSessionWebSocket, updateAgent, agents]);

  const handleCreateAgent = async (config: CreateAgentRequest) => {
    try {
      const response = await agentsApi.create(config);
      updateAgent(response.data);
      setCreateDialogOpen(false);

      if (config.auto_join) {
        try {
          await agentsApi.start(response.data.id);
          addNotification({
            type: 'success',
            title: 'Agent created and started',
            message: `Agent "${response.data.config.name}" has been created and started successfully.`,
          });
        } catch {
          addNotification({
            type: 'warning',
            title: 'Agent created but failed to start',
            message: `Agent "${response.data.config.name}" was created but failed to start automatically.`,
          });
        }
      } else {
        addNotification({
          type: 'success',
          title: 'Agent created',
          message: `Agent "${response.data.config.name}" has been created successfully.`,
        });
      }
    } catch (error) {
      console.error('Failed to create agent:', error);
      addNotification({
        type: 'error',
        title: 'Failed to create agent',
        message: 'Could not create agent. Please check your configuration.',
      });
    }
  };

  const handleStartAgent = async (agentId: string) => {
    try {
      await agentsApi.start(agentId);
      addNotification({
        type: 'success',
        title: 'Agent started',
        message: 'Agent has been started successfully.',
      });
    } catch (error) {
      console.error('Failed to start agent:', error);
      addNotification({
        type: 'error',
        title: 'Failed to start agent',
        message: 'Could not start agent. Please try again.',
      });
    }
  };

  const handleStopAgent = async (agentId: string) => {
    try {
      await agentsApi.stop(agentId);
      addNotification({
        type: 'success',
        title: 'Agent stopped',
        message: 'Agent has been stopped successfully.',
      });
    } catch (error) {
      console.error('Failed to stop agent:', error);
      addNotification({
        type: 'error',
        title: 'Failed to stop agent',
        message: 'Could not stop agent. Please try again.',
      });
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'running':
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      case 'error':
        return <AlertCircle className="h-4 w-4 text-red-600" />;
      case 'starting':
      case 'stopping':
        return <Loader2 className="h-4 w-4 text-blue-600 animate-spin" />;
      default:
        return <Square className="h-4 w-4 text-gray-400" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
      case 'error':
        return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300';
      case 'starting':
      case 'stopping':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
    }
  };

  return (
    <Layout title="Dashboard" subtitle="Monitor and manage your AI agents">
      <div className="space-y-8">
        {/* Header */}
        <div className="flex justify-between items-center">
          <div>
            <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
              <p className="text-muted-foreground">
                Welcome back! Here&apos;s an overview of your AI agents and meetings.
              </p>
          </div>
          <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
            <DialogTrigger asChild>
              <Button size="lg" className="gap-2">
                <Plus className="h-5 w-5" />
                Create Agent
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-2xl">
              <DialogHeader>
                <DialogTitle>Create New Agent</DialogTitle>
              </DialogHeader>
              <CreateAgentPage onCreate={handleCreateAgent} onCancel={() => setCreateDialogOpen(false)} />
            </DialogContent>
          </Dialog>
        </div>

        {/* Stats Cards */}
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Agents</CardTitle>
              <Bot className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.totalAgents}</div>
              <p className="text-xs text-muted-foreground">
                {stats.runningAgents} running
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Active Agents</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-600">{stats.runningAgents}</div>
              <p className="text-xs text-muted-foreground">
                Currently in meetings
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Meetings</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.totalMeetings}</div>
              <p className="text-xs text-muted-foreground">
                Active sessions
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Avg. Uptime</CardTitle>
              <TrendingUp className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.avgUptime}%</div>
              <p className="text-xs text-muted-foreground">
                Last 24 hours
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Recent Agents */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Recent Agents</CardTitle>
                <CardDescription>
                  Your most recently created agents
                </CardDescription>
              </div>
              <Button variant="outline" size="sm">
                View All
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="flex justify-center items-center py-8">
                <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
              </div>
            ) : agents.length === 0 ? (
              <div className="text-center py-12">
                <Bot className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <h3 className="text-lg font-medium mb-2">No agents yet</h3>
                <p className="text-muted-foreground mb-4">
                  Create your first AI agent to get started with automated meeting participation.
                </p>
                <Button onClick={() => setCreateDialogOpen(true)}>
                  <Plus className="h-4 w-4 mr-2" />
                  Create Your First Agent
                </Button>
              </div>
            ) : (
              <div className="space-y-4">
                {agents.slice(0, 5).map((agent) => (
                  <div key={agent.id} className="flex items-center justify-between p-4 border rounded-lg">
                    <div className="flex items-center gap-4">
                      <div className="flex items-center gap-2">
                        {getStatusIcon(agent.status)}
                        <div>
                          <p className="font-medium">{agent.config.name}</p>
                          <p className="text-sm text-muted-foreground">
                            {new URL(agent.config.meeting_url).hostname}
                          </p>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      <Badge className={getStatusColor(agent.status)}>
                        {agent.status}
                      </Badge>
                      <div className="flex gap-2">
                        {agent.status === 'running' ? (
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleStopAgent(agent.id)}
                          >
                            <Square className="h-4 w-4" />
                          </Button>
                        ) : agent.status === 'stopped' || agent.status === 'error' ? (
                          <Button
                            size="sm"
                            onClick={() => handleStartAgent(agent.id)}
                          >
                            <Play className="h-4 w-4" />
                          </Button>
                        ) : (
                          <Button size="sm" variant="outline" disabled>
                            <Loader2 className="h-4 w-4 animate-spin" />
                          </Button>
                        )}
                        <Button size="sm" variant="outline">
                          <Settings className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Quick Actions */}
        <Card>
          <CardHeader>
            <CardTitle>Quick Actions</CardTitle>
            <CardDescription>
              Common tasks and shortcuts
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              <Button variant="outline" className="h-20 flex-col gap-2">
                <Plus className="h-6 w-6" />
                <span className="text-sm">Create Agent</span>
              </Button>
              <Button variant="outline" className="h-20 flex-col gap-2">
                <Bot className="h-6 w-6" />
                <span className="text-sm">Manage Agents</span>
              </Button>
              <Button variant="outline" className="h-20 flex-col gap-2">
                <Activity className="h-6 w-6" />
                <span className="text-sm">View Logs</span>
              </Button>
              <Button variant="outline" className="h-20 flex-col gap-2">
                <Settings className="h-6 w-6" />
                <span className="text-sm">Settings</span>
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}