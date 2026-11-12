import { AGE_RANGES, QUESTIONS } from '../../data.js';

// Submit explicit questionnaire selections only. Voice, notifications, referrals,
// survey answers and arbitrary React state are not conversational background.
const CONTEXT_QUESTIONS = [
  'identity', 'therapist', 'concerns', 'focus', 'feelings', 'faith',
  'neurodivergent', 'relationships', 'interests', 'strengths', 'barriers', 'recharge',
];

export function buildOnboardingAnswers(answers = {}) {
  const result = {};
  if (AGE_RANGES.includes(answers.age)) result.age = answers.age;
  if (Number.isInteger(answers.mood) && answers.mood >= 0 && answers.mood <= 4) result.mood = answers.mood;
  for (const key of CONTEXT_QUESTIONS) {
    const question = QUESTIONS[key];
    const value = answers[key];
    if (question.type === 'single') {
      if (question.options.includes(value)) result[key] = value;
    } else if (Array.isArray(value)) {
      result[key] = [...new Set(value.filter(item => question.options.includes(item)))].slice(0, question.limit ?? 8);
    }
  }
  return result;
}
