'use client';

import { useState, useCallback } from 'react';
import { UseFormRegister, FieldErrors } from 'react-hook-form';
import {
  FileText,
  Lightbulb,
  Copy,
  BookOpen,
  Settings,
  User,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { AlertCircle } from 'lucide-react';

interface PromptConfigurationProps {
  conversationMode: 'conversational' | 'analyst';
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  register: UseFormRegister<any>;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  errors: FieldErrors<any>;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  watch: (name: string) => any;
}

const CONVERSATIONAL_PROMPT_FIELD = {
  key: 'custom_prompt',
  label: 'Response Prompt',
  icon: Settings,
  description: 'Customize how your agent generates responses to participants',
  placeholder: 'You are {agent_name}, a helpful assistant. The participant {speaker} just said: "{text}". Respond naturally and helpfully based on the context: {context}',
  examples: [
    'You are {agent_name}, a professional meeting facilitator. Respond concisely and keep discussions on track.',
    'You are {agent_name}, a technical expert. Provide detailed explanations and ask clarifying questions.',
    'You are {agent_name}, a supportive coach. Focus on encouragement and positive reinforcement.',
  ],
  tips: [
    'Use {agent_name} to maintain consistent identity',
    'Reference {speaker} and {text} for contextual responses',
    'Include {context} for conversation history awareness',
    'Keep prompts focused on your agent\'s role and expertise',
  ],
  required: false,
};

// Removed ANALYST_PROMPT_FIELDS - now using personality-driven prompts

export function PromptConfiguration({
  conversationMode,
  register,
  watch,
}: PromptConfigurationProps) {
  const [showExamples, setShowExamples] = useState(false);

  const copyToClipboard = useCallback(async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
    } catch (err) {
      console.error('Failed to copy text: ', err);
    }
  }, []);

  const insertPlaceholder = useCallback((fieldKey: string, placeholder: string) => {
    const textarea = document.querySelector(`textarea[name="${fieldKey}"]`) as HTMLTextAreaElement;
    if (textarea) {
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      const text = textarea.value;
      const newText = text.substring(0, start) + placeholder + text.substring(end);
      textarea.value = newText;
      textarea.focus();
      textarea.setSelectionRange(start + placeholder.length, start + placeholder.length);
    }
  }, []);

  if (conversationMode === 'conversational') {
    return (
      <div className="space-y-6">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <Label className="text-sm font-medium flex items-center gap-2">
                <Settings className="h-4 w-4" />
                Response Prompt
              </Label>
              <p className="text-xs text-slate-500 dark:text-zinc-500">
                Customize how your agent generates responses to participants
              </p>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setShowExamples(!showExamples)}
              className="text-xs"
            >
              <Lightbulb className="h-3 w-3 mr-1" />
              {showExamples ? 'Hide' : 'Show'} Examples
            </Button>
          </div>

          <Textarea
            {...register(CONVERSATIONAL_PROMPT_FIELD.key)}
            placeholder={CONVERSATIONAL_PROMPT_FIELD.placeholder}
            rows={6}
            className="font-mono text-sm"
          />

          {showExamples && (
            <Card className="bg-slate-50 dark:bg-slate-900/50">
              <CardHeader className="pb-3">
                <CardTitle className="text-sm flex items-center gap-2">
                  <Lightbulb className="h-4 w-4" />
                  Example Prompts
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {CONVERSATIONAL_PROMPT_FIELD.examples.map((example, index) => (
                  <div key={index} className="flex items-start gap-2 p-2 rounded border bg-white dark:bg-slate-800">
                    <div className="flex-1 text-xs font-mono text-slate-700 dark:text-slate-300">
                      {example}
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => copyToClipboard(example)}
                      className="h-6 w-6 p-0"
                    >
                      <Copy className="h-3 w-3" />
                    </Button>
                  </div>
                ))}
              </CardContent>
            </Card>
          )}

          <div className="space-y-2">
            <Label className="text-xs font-medium text-slate-600 dark:text-zinc-400">
              Available Placeholders
            </Label>
            <div className="flex flex-wrap gap-1">
              {['{agent_name}', '{speaker}', '{text}', '{context}'].map((placeholder) => (
                <Badge
                  key={placeholder}
                  variant="outline"
                  className="text-xs cursor-pointer hover:bg-slate-100 dark:hover:bg-slate-800"
                  onClick={() => insertPlaceholder(CONVERSATIONAL_PROMPT_FIELD.key, placeholder)}
                >
                  {placeholder}
                </Badge>
              ))}
            </div>
          </div>

          <Alert>
            <AlertCircle className="h-4 w-4" />
            <AlertDescription className="text-xs">
              Leave empty to use the default conversational prompt. Placeholders will be automatically replaced with actual values during conversations.
            </AlertDescription>
          </Alert>
        </div>
      </div>
    );
  }

  if (conversationMode === 'analyst') {
    return (
      <div className="space-y-6">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <Label className="text-sm font-medium flex items-center gap-2">
                <User className="h-4 w-4" />
                Agent Personality
              </Label>
              <p className="text-xs text-slate-500 dark:text-zinc-500">
                Define your agent&apos;s personality and expertise. This will be used to generate tailored prompts for each analysis task.
              </p>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setShowExamples(!showExamples)}
              className="text-xs"
            >
              <BookOpen className="h-3 w-3 mr-1" />
              {showExamples ? 'Hide Templates' : 'Show Templates'}
            </Button>
          </div>
          <Textarea
            {...register('personality_prompt')}
            placeholder="You are a seasoned business analyst with 10+ years of experience in corporate strategy and meeting facilitation. You excel at identifying key business insights, action items, and strategic opportunities from discussions. Your analysis is thorough, actionable, and focused on driving business outcomes."
            className="min-h-[150px] font-mono text-sm resize-none"
          />
          {showExamples && (
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400">
                <BookOpen className="h-3 w-3" />
                <span>Click a personality template to fill the field</span>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <Card
                  className="cursor-pointer hover:shadow-md transition-shadow border-blue-200 dark:border-blue-800 bg-blue-50/50 dark:bg-blue-950/50"
                  onClick={() => {
                    const textarea = document.querySelector('textarea[name="personality_prompt"]') as HTMLTextAreaElement;
                    if (textarea) {
                      textarea.value = "You are a seasoned business analyst with 10+ years of experience in corporate strategy and meeting facilitation. You excel at identifying key business insights, action items, and strategic opportunities from discussions. Your analysis is thorough, actionable, and focused on driving business outcomes.";
                    }
                  }}
                >
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <Badge variant="secondary" className="text-xs bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">Business Analyst</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="pt-0">
                    <p className="text-xs text-slate-600 dark:text-slate-400">Perfect for business meetings, strategy sessions, and executive discussions.</p>
                  </CardContent>
                </Card>
                <Card
                  className="cursor-pointer hover:shadow-md transition-shadow border-green-200 dark:border-green-800 bg-green-50/50 dark:bg-green-950/50"
                  onClick={() => {
                    const textarea = document.querySelector('textarea[name="personality_prompt"]') as HTMLTextAreaElement;
                    if (textarea) {
                      textarea.value = "You are a professional legal counsel specializing in contract analysis and compliance. With extensive experience in corporate law, you focus on identifying legal implications, contractual obligations, and regulatory requirements from business discussions. Your analysis emphasizes risk mitigation and legal best practices.";
                    }
                  }}
                >
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <Badge variant="secondary" className="text-xs bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">Legal Counsel</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="pt-0">
                    <p className="text-xs text-slate-600 dark:text-slate-400">Ideal for legal discussions, contract negotiations, and compliance meetings.</p>
                  </CardContent>
                </Card>
                <Card
                  className="cursor-pointer hover:shadow-md transition-shadow border-purple-200 dark:border-purple-800 bg-purple-50/50 dark:bg-purple-950/50"
                  onClick={() => {
                    const textarea = document.querySelector('textarea[name="personality_prompt"]') as HTMLTextAreaElement;
                    if (textarea) {
                      textarea.value = "You are a technical project manager with deep expertise in software development and agile methodologies. You specialize in identifying technical challenges, project risks, and implementation details from technical discussions. Your analysis focuses on development best practices, timeline management, and technical decision-making.";
                    }
                  }}
                >
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <Badge variant="secondary" className="text-xs bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200">Tech Project Manager</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="pt-0">
                    <p className="text-xs text-slate-600 dark:text-slate-400">Great for development meetings, sprint planning, and technical reviews.</p>
                  </CardContent>
                </Card>
                <Card
                  className="cursor-pointer hover:shadow-md transition-shadow border-orange-200 dark:border-orange-800 bg-orange-50/50 dark:bg-orange-950/50"
                  onClick={() => {
                    const textarea = document.querySelector('textarea[name="personality_prompt"]') as HTMLTextAreaElement;
                    if (textarea) {
                      textarea.value = "You are an HR professional with expertise in team dynamics, employee engagement, and organizational development. You focus on interpersonal aspects, team morale, communication patterns, and organizational health indicators from workplace discussions. Your analysis emphasizes people-first approaches and team building.";
                    }
                  }}
                >
                  <CardHeader className="pb-2">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <Badge variant="secondary" className="text-xs bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200">HR Specialist</Badge>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="pt-0">
                    <p className="text-xs text-slate-600 dark:text-slate-400">Perfect for team meetings, performance reviews, and organizational discussions.</p>
                  </CardContent>
                </Card>
              </div>
              <Alert className="border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-950">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription className="text-xs">
                  The personality description will be used to automatically generate appropriate prompts for different analysis tasks (summary, key points, action items, etc.). Be specific about the agent&apos;s expertise, experience level, and analytical focus.
                </AlertDescription>
              </Alert>
            </div>
          )}
        </div>
      </div>
    );
  }

  return null;
}
