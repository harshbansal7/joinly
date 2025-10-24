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

const CONVERSATIONAL_EXAMPLES = [
  'You are {agent_name}, a professional meeting facilitator. Respond concisely and keep discussions on track.',
  'You are {agent_name}, a technical expert. Provide detailed explanations and ask clarifying questions.',
  'You are {agent_name}, a supportive coach. Focus on encouragement and positive reinforcement.',
];

const ANALYST_TEMPLATES = [
  {
    name: 'Business Analyst',
    description: 'Perfect for business meetings, strategy sessions, and executive discussions.',
    prompt: 'You are a seasoned business analyst with 10+ years of experience in corporate strategy and meeting facilitation. You excel at identifying key business insights, action items, and strategic opportunities from discussions. Your analysis is thorough, actionable, and focused on driving business outcomes.',
    color: 'blue'
  },
  {
    name: 'Legal Counsel',
    description: 'Ideal for legal discussions, contract negotiations, and compliance meetings.',
    prompt: 'You are a professional legal counsel specializing in contract analysis and compliance. With extensive experience in corporate law, you focus on identifying legal implications, contractual obligations, and regulatory requirements from business discussions. Your analysis emphasizes risk mitigation and legal best practices.',
    color: 'green'
  },
  {
    name: 'Tech Project Manager',
    description: 'Great for development meetings, sprint planning, and technical reviews.',
    prompt: 'You are a technical project manager with deep expertise in software development and agile methodologies. You specialize in identifying technical challenges, project risks, and implementation details from technical discussions. Your analysis focuses on development best practices, timeline management, and technical decision-making.',
    color: 'purple'
  },
  {
    name: 'HR Specialist',
    description: 'Perfect for team meetings, performance reviews, and organizational discussions.',
    prompt: 'You are an HR professional with expertise in team dynamics, employee engagement, and organizational development. You focus on interpersonal aspects, team morale, communication patterns, and organizational health indicators from workplace discussions. Your analysis emphasizes people-first approaches and team building.',
    color: 'orange'
  }
];

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

  const insertPlaceholder = useCallback((placeholder: string) => {
    const textarea = document.querySelector('textarea[name="custom_prompt"]') as HTMLTextAreaElement;
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

  const setTemplatePrompt = useCallback((prompt: string) => {
    const textarea = document.querySelector('textarea[name="custom_prompt"]') as HTMLTextAreaElement;
    if (textarea) {
      textarea.value = prompt;
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
            {...register('custom_prompt')}
            placeholder="You are {agent_name}, a helpful assistant. The participant {speaker} just said: '{text}'. Respond naturally and helpfully based on the context: {context}"
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
                {CONVERSATIONAL_EXAMPLES.map((example, index) => (
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
                  onClick={() => insertPlaceholder(placeholder)}
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
                Agent Role & Expertise
              </Label>
              <p className="text-xs text-slate-500 dark:text-zinc-500">
                Define your agent&apos;s role and expertise. This will be used to generate tailored prompts for each analysis task.
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
            {...register('custom_prompt')}
            placeholder="You are a seasoned business analyst with 10+ years of experience in corporate strategy and meeting facilitation. You excel at identifying key business insights, action items, and strategic opportunities from discussions. Your analysis is thorough, actionable, and focused on driving business outcomes."
            className="min-h-[150px] font-mono text-sm resize-none"
          />
          {showExamples && (
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400">
                <BookOpen className="h-3 w-3" />
                <span>Click a role template to fill the field</span>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                {ANALYST_TEMPLATES.map((template, index) => (
                  <Card
                    key={index}
                    className={`cursor-pointer hover:shadow-md transition-shadow border-${template.color}-200 dark:border-${template.color}-800 bg-${template.color}-50/50 dark:bg-${template.color}-950/50`}
                    onClick={() => setTemplatePrompt(template.prompt)}
                  >
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm flex items-center gap-2">
                        <Badge variant="secondary" className={`text-xs bg-${template.color}-100 text-${template.color}-800 dark:bg-${template.color}-900 dark:text-${template.color}-200`}>
                          {template.name}
                        </Badge>
                      </CardTitle>
                    </CardHeader>
                    <CardContent className="pt-0">
                      <p className="text-xs text-slate-600 dark:text-slate-400">{template.description}</p>
                    </CardContent>
                  </Card>
                ))}
              </div>
              <Alert className="border-amber-200 dark:border-amber-800 bg-amber-50 dark:bg-amber-950">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription className="text-xs">
                  The role description will be used to automatically generate appropriate prompts for different analysis tasks (summary, key points, action items, etc.). Be specific about the agent&apos;s expertise, experience level, and analytical focus.
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
