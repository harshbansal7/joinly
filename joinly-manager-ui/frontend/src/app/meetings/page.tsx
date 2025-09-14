/**
 * Meetings management page.
 */

'use client';

import { useEffect, useCallback } from 'react';
import { ExternalLink, Users } from 'lucide-react';

import { Layout } from '@/components/layout/layout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useAgentStore } from '@/lib/store';
import { meetingsApi } from '@/lib/api';

export default function MeetingsPage() {
  const { meetings, isLoading, setMeetings, setLoading } = useAgentStore();

  const loadMeetings = useCallback(async () => {
    setLoading(true);
    try {
      const response = await meetingsApi.list();
      setMeetings(response.data);
    } catch (error) {
      console.error('Failed to load meetings:', error);
    } finally {
      setLoading(false);
    }
  }, [setMeetings, setLoading]);

  useEffect(() => {
    loadMeetings();
  }, [loadMeetings]);

  return (
    <Layout title="Meetings" subtitle="Active meetings with agents">
      <div className="space-y-6">
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Meetings</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{meetings.length}</div>
              <p className="text-xs text-muted-foreground">
                With agents present
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Agents</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {meetings.reduce((sum, meeting) => sum + meeting.agent_count, 0)}
              </div>
              <p className="text-xs text-muted-foreground">
                Across all meetings
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Avg Agents/Meeting</CardTitle>
              <Users className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {meetings.length > 0
                  ? (meetings.reduce((sum, meeting) => sum + meeting.agent_count, 0) / meetings.length).toFixed(1)
                  : '0'
                }
              </div>
              <p className="text-xs text-muted-foreground">
                Average distribution
              </p>
            </CardContent>
          </Card>
        </div>

        {/* Meetings Grid */}
        {isLoading ? (
          <div className="flex justify-center items-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
          </div>
        ) : meetings.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-12">
              <div className="text-center">
                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                  No active meetings
                </h3>
                <p className="text-gray-600 dark:text-gray-400 mb-4">
                  Meetings will appear here when agents are active
                </p>
                <Button onClick={loadMeetings}>
                  Refresh
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {meetings.map((meeting) => (
              <Card key={meeting.url}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg truncate" title={meeting.url}>
                      {new URL(meeting.url).hostname}
                    </CardTitle>
                    <Badge variant="outline">
                      {meeting.agent_count} agent{meeting.agent_count !== 1 ? 's' : ''}
                    </Badge>
                  </div>
                  <CardDescription>
                    Started {new Date(meeting.created_at).toLocaleString()}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    <div className="text-sm text-gray-600 dark:text-gray-400">
                      <div>Meeting ID: {new URL(meeting.url).pathname.slice(1)}</div>
                      <div>Agents: {meeting.agent_ids.join(', ')}</div>
                    </div>

                    <Button
                      variant="outline"
                      size="sm"
                      className="w-full"
                      onClick={() => window.open(meeting.url, '_blank')}
                    >
                      <ExternalLink className="h-4 w-4 mr-2" />
                      Open Meeting
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </Layout>
  );
}
