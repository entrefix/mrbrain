import {
  addDays,
  addWeeks,
  addMonths,
  addHours,
  addMinutes,
  nextMonday,
  nextFriday,
  nextSaturday,
  setHours,
  setMinutes,
  format,
  startOfDay,
  endOfMonth,
  setDate,
  setMonth,
  setYear,
  getDay
} from 'date-fns';

export interface ParsedDate {
  date: Date | null;
  matchedText: string;
  startIndex: number;
  endIndex: number;
}

export interface DateMatch {
  pattern: RegExp;
  parser: (match: RegExpMatchArray) => Date;
}

// Month name to number mapping
const monthMap: Record<string, number> = {
  jan: 0, january: 0,
  feb: 1, february: 1,
  mar: 2, march: 2,
  apr: 3, april: 3,
  may: 4,
  jun: 5, june: 5,
  jul: 6, july: 6,
  aug: 7, august: 7,
  sep: 8, sept: 8, september: 8,
  oct: 9, october: 9,
  nov: 10, november: 10,
  dec: 11, december: 11,
};

// Day name to number mapping (0 = Sunday)
const dayMap: Record<string, number> = {
  sunday: 0, sun: 0,
  monday: 1, mon: 1,
  tuesday: 2, tue: 2,
  wednesday: 3, wed: 3,
  thursday: 4, thu: 4,
  friday: 5, fri: 5,
  saturday: 6, sat: 6,
};

// Helper to parse time from matches
function parseTime(hoursStr: string | undefined, minutesStr: string | undefined, meridiem: string | undefined): { hours: number; minutes: number } {
  if (!hoursStr) return { hours: 0, minutes: 0 };

  let hours = parseInt(hoursStr);
  const minutes = minutesStr ? parseInt(minutesStr) : 0;
  const m = meridiem?.toLowerCase();

  if (m === 'pm' && hours < 12) hours += 12;
  if (m === 'am' && hours === 12) hours = 0;
  // If no meridiem and hour is small (1-6), assume PM
  if (!m && hours >= 1 && hours <= 6) hours += 12;

  return { hours, minutes };
}

// Helper to get next occurrence of a weekday
function getNextWeekday(dayName: string, skipThisWeek: boolean = false): Date {
  const today = new Date();
  const targetDay = dayMap[dayName.toLowerCase()];
  const currentDay = today.getDay();
  let daysToAdd = targetDay - currentDay;

  if (daysToAdd <= 0 || skipThisWeek) {
    daysToAdd += 7;
  }
  if (skipThisWeek && daysToAdd <= 7) {
    daysToAdd += 7;
  }

  return addDays(startOfDay(today), daysToAdd);
}

// Helper to apply time to a date
function applyTime(date: Date, hours: number, minutes: number): Date {
  return setMinutes(setHours(date, hours), minutes);
}

// Natural language date patterns (sorted by specificity - more specific first)
const datePatterns: DateMatch[] = [
  // ============================================
  // CATEGORY 1: Specific Calendar Dates with Time
  // ============================================

  // "jan 15 at 3pm", "january 15th at 9:30am", "jan 15, 2025 at 5pm"
  {
    pattern: /\b(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\s+(\d{1,2})(?:st|nd|rd|th)?(?:,?\s+(\d{4}))?\s+(?:at\s+)?(\d{1,2})(?::(\d{2}))?\s*(am|pm)/i,
    parser: (match) => {
      const monthName = match[1].toLowerCase();
      const day = parseInt(match[2]);
      const year = match[3] ? parseInt(match[3]) : new Date().getFullYear();
      const { hours, minutes } = parseTime(match[4], match[5], match[6]);

      let date = new Date(year, monthMap[monthName], day);
      return applyTime(date, hours, minutes);
    },
  },

  // "jan 15", "january 15th", "jan 15, 2025"
  {
    pattern: /\b(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\s+(\d{1,2})(?:st|nd|rd|th)?(?:,?\s+(\d{4}))?\b/i,
    parser: (match) => {
      const monthName = match[1].toLowerCase();
      const day = parseInt(match[2]);
      const year = match[3] ? parseInt(match[3]) : new Date().getFullYear();

      return startOfDay(new Date(year, monthMap[monthName], day));
    },
  },

  // "15 jan", "15th january"
  {
    pattern: /\b(\d{1,2})(?:st|nd|rd|th)?\s+(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\b/i,
    parser: (match) => {
      const day = parseInt(match[1]);
      const monthName = match[2].toLowerCase();
      const year = new Date().getFullYear();

      return startOfDay(new Date(year, monthMap[monthName], day));
    },
  },

  // "1/15", "01/15", "1/15/2025" (US format: month/day)
  {
    pattern: /\b(\d{1,2})\/(\d{1,2})(?:\/(\d{4}))?\b/,
    parser: (match) => {
      const month = parseInt(match[1]) - 1; // 0-indexed
      const day = parseInt(match[2]);
      const year = match[3] ? parseInt(match[3]) : new Date().getFullYear();

      return startOfDay(new Date(year, month, day));
    },
  },

  // ============================================
  // CATEGORY 2: Weekdays with Time
  // ============================================

  // "next monday 9am", "next tuesday at 5pm"
  {
    pattern: /\bnext\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)(?:\s+(?:at\s+)?(\d{1,2})(?::(\d{2}))?\s*(am|pm)?)?/i,
    parser: (match) => {
      const dayName = match[1].toLowerCase();
      let date = getNextWeekday(dayName, true);

      if (match[2]) {
        const { hours, minutes } = parseTime(match[2], match[3], match[4]);
        date = applyTime(date, hours, minutes);
      }

      return date;
    },
  },

  // "this monday 9am", "this friday at 3pm"
  {
    pattern: /\bthis\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)(?:\s+(?:at\s+)?(\d{1,2})(?::(\d{2}))?\s*(am|pm)?)?/i,
    parser: (match) => {
      const dayName = match[1].toLowerCase();
      const today = new Date();
      const targetDay = dayMap[dayName];
      const currentDay = today.getDay();
      let daysToAdd = targetDay - currentDay;
      if (daysToAdd < 0) daysToAdd += 7;

      let date = addDays(startOfDay(today), daysToAdd);

      if (match[2]) {
        const { hours, minutes } = parseTime(match[2], match[3], match[4]);
        date = applyTime(date, hours, minutes);
      }

      return date;
    },
  },

  // "on monday 9am", "on friday at 3pm"
  {
    pattern: /\bon\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)(?:\s+(?:at\s+)?(\d{1,2})(?::(\d{2}))?\s*(am|pm)?)?/i,
    parser: (match) => {
      const dayName = match[1].toLowerCase();
      let date = getNextWeekday(dayName);

      if (match[2]) {
        const { hours, minutes } = parseTime(match[2], match[3], match[4]);
        date = applyTime(date, hours, minutes);
      }

      return date;
    },
  },

  // "coming monday", "coming friday"
  {
    pattern: /\bcoming\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b/i,
    parser: (match) => {
      const dayName = match[1].toLowerCase();
      return getNextWeekday(dayName);
    },
  },

  // ============================================
  // CATEGORY 3: Tomorrow/Today with Time
  // ============================================

  // "tomorrow 9am", "tomorrow at 5pm", "tmr at 3pm"
  {
    pattern: /\b(tomorrow|tmr|tmrw|tom)(?:\s+(?:at\s+)?(\d{1,2})(?::(\d{2}))?\s*(am|pm)?)?/i,
    parser: (match) => {
      let date = addDays(startOfDay(new Date()), 1);

      if (match[2]) {
        const { hours, minutes } = parseTime(match[2], match[3], match[4]);
        date = applyTime(date, hours, minutes);
      }

      return date;
    },
  },

  // "today at 3pm", "today 5pm"
  {
    pattern: /\btoday(?:\s+(?:at\s+)?(\d{1,2})(?::(\d{2}))?\s*(am|pm)?)?/i,
    parser: (match) => {
      let date = new Date();

      if (match[1]) {
        const { hours, minutes } = parseTime(match[1], match[2], match[3]);
        date = applyTime(date, hours, minutes);
      } else {
        date = startOfDay(date);
      }

      return date;
    },
  },

  // ============================================
  // CATEGORY 4: Relative Time Periods
  // ============================================

  // "in X hours"
  {
    pattern: /\bin\s+(\d+)\s+hours?\b/i,
    parser: (match) => addHours(new Date(), parseInt(match[1])),
  },

  // "in X minutes"
  {
    pattern: /\bin\s+(\d+)\s+minutes?\b/i,
    parser: (match) => addMinutes(new Date(), parseInt(match[1])),
  },

  // "in X days"
  {
    pattern: /\bin\s+(\d+)\s+days?\b/i,
    parser: (match) => addDays(startOfDay(new Date()), parseInt(match[1])),
  },

  // "in X weeks"
  {
    pattern: /\bin\s+(\d+)\s+weeks?\b/i,
    parser: (match) => addWeeks(startOfDay(new Date()), parseInt(match[1])),
  },

  // "in X months"
  {
    pattern: /\bin\s+(\d+)\s+months?\b/i,
    parser: (match) => addMonths(startOfDay(new Date()), parseInt(match[1])),
  },

  // "in a week", "in a month", "in a day", "in an hour"
  {
    pattern: /\bin\s+(?:a|an)\s+(week|month|day|hour)\b/i,
    parser: (match) => {
      const unit = match[1].toLowerCase();
      const now = new Date();
      switch (unit) {
        case 'hour': return addHours(now, 1);
        case 'day': return addDays(startOfDay(now), 1);
        case 'week': return addWeeks(startOfDay(now), 1);
        case 'month': return addMonths(startOfDay(now), 1);
        default: return now;
      }
    },
  },

  // "next week"
  {
    pattern: /\bnext\s+week\b/i,
    parser: () => nextMonday(startOfDay(new Date())),
  },

  // "next month"
  {
    pattern: /\bnext\s+month\b/i,
    parser: () => {
      const today = new Date();
      return startOfDay(setDate(addMonths(today, 1), 1));
    },
  },

  // ============================================
  // CATEGORY 5: Weekends
  // ============================================

  // "this weekend"
  {
    pattern: /\bthis\s+weekend\b/i,
    parser: () => nextSaturday(startOfDay(new Date())),
  },

  // "next weekend"
  {
    pattern: /\bnext\s+weekend\b/i,
    parser: () => addWeeks(nextSaturday(startOfDay(new Date())), 1),
  },

  // ============================================
  // CATEGORY 6: Special Cases
  // ============================================

  // "end of month" / "eom"
  {
    pattern: /\b(end\s+of\s+month|eom)\b/i,
    parser: () => endOfMonth(new Date()),
  },

  // "end of week" / "eow"
  {
    pattern: /\b(end\s+of\s+week|eow)\b/i,
    parser: () => nextFriday(startOfDay(new Date())),
  },

  // "end of day" / "eod"
  {
    pattern: /\b(end\s+of\s+day|eod)\b/i,
    parser: () => setMinutes(setHours(new Date(), 17), 0), // 5 PM today
  },

  // "tonight"
  {
    pattern: /\btonight\b/i,
    parser: () => setMinutes(setHours(new Date(), 20), 0), // 8 PM today
  },

  // "yesterday"
  {
    pattern: /\byesterday\b/i,
    parser: () => addDays(startOfDay(new Date()), -1),
  },

  // ============================================
  // CATEGORY 7: Standalone Weekdays (must be after more specific patterns)
  // ============================================

  // Standalone full weekday names: "monday", "friday" etc.
  {
    pattern: /\b(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b/i,
    parser: (match) => {
      const dayName = match[1].toLowerCase();
      return getNextWeekday(dayName);
    },
  },

  // Abbreviated weekdays: "mon", "tue", "wed", etc.
  {
    pattern: /\b(mon|tue|wed|thu|fri|sat|sun)\b/i,
    parser: (match) => {
      const dayName = match[1].toLowerCase();
      return getNextWeekday(dayName);
    },
  },

  // ============================================
  // CATEGORY 8: Special Time Keywords
  // ============================================

  // "noon" / "midday"
  {
    pattern: /\b(noon|midday)\b/i,
    parser: () => setMinutes(setHours(new Date(), 12), 0),
  },

  // "midnight"
  {
    pattern: /\bmidnight\b/i,
    parser: () => setMinutes(setHours(addDays(new Date(), 1), 0), 0), // Midnight = start of next day
  },

  // "morning"
  {
    pattern: /\bmorning\b/i,
    parser: () => setMinutes(setHours(new Date(), 9), 0), // 9 AM
  },

  // "afternoon"
  {
    pattern: /\bafternoon\b/i,
    parser: () => setMinutes(setHours(new Date(), 14), 0), // 2 PM
  },

  // "evening"
  {
    pattern: /\bevening\b/i,
    parser: () => setMinutes(setHours(new Date(), 18), 0), // 6 PM
  },

  // "night"
  {
    pattern: /\bnight\b/i,
    parser: () => setMinutes(setHours(new Date(), 21), 0), // 9 PM
  },

  // ============================================
  // CATEGORY 9: Standalone Time (last - most generic)
  // ============================================

  // Standalone time "9pm", "10:30am", "at 5pm" (assumes today)
  {
    pattern: /\b(?:at\s+)?(\d{1,2})(?::(\d{2}))?\s*(am|pm)\b/i,
    parser: (match) => {
      const { hours, minutes } = parseTime(match[1], match[2], match[3]);
      return applyTime(new Date(), hours, minutes);
    },
  },
];

/**
 * Parse natural language date from text
 * Returns the parsed date and the matched text range for highlighting
 */
export function parseDateFromText(text: string): ParsedDate | null {
  for (const { pattern, parser } of datePatterns) {
    const match = text.match(pattern);
    if (match && match.index !== undefined) {
      try {
        const date = parser(match);
        return {
          date,
          matchedText: match[0],
          startIndex: match.index,
          endIndex: match.index + match[0].length,
        };
      } catch {
        // If parsing fails, continue to next pattern
        continue;
      }
    }
  }
  return null;
}

/**
 * Extract the title without the date portion
 */
export function extractTitleWithoutDate(text: string): string {
  const parsed = parseDateFromText(text);
  if (!parsed) return text;

  // Remove the matched date text and clean up extra spaces
  const before = text.substring(0, parsed.startIndex);
  const after = text.substring(parsed.endIndex);
  return (before + after).replace(/\s+/g, ' ').trim();
}

/**
 * Format a date for display
 */
export function formatDateForDisplay(date: Date): string {
  const today = startOfDay(new Date());
  const targetDate = startOfDay(date);
  const diffDays = Math.round((targetDate.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));

  // Check if date has a specific time (not midnight)
  const hasTime = date.getHours() !== 0 || date.getMinutes() !== 0;
  const timeStr = hasTime ? ` at ${format(date, 'h:mm a')}` : '';

  if (diffDays === 0) return 'Today' + timeStr;
  if (diffDays === 1) return 'Tomorrow' + timeStr;
  if (diffDays === -1) return 'Yesterday' + timeStr;
  if (diffDays > 1 && diffDays <= 7) {
    return format(date, 'EEEE') + timeStr; // Day name
  }
  return format(date, 'MMM d') + timeStr;
}

/**
 * Get date urgency level for todo due dates
 */
export function getDateUrgency(dueDate: string | null): 'overdue' | 'today' | 'this-week' | 'future' | null {
  if (!dueDate) return null;
  const date = new Date(dueDate);
  const today = startOfDay(new Date());
  const diffDays = Math.round((date.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
  
  if (diffDays < 0) return 'overdue';
  if (diffDays === 0) return 'today';
  if (diffDays <= 7) return 'this-week';
  return 'future';
}

/**
 * Check if a date is overdue
 */
export function isOverdue(date: string | null): boolean {
  if (!date) return false;
  const dueDate = new Date(date);
  const today = startOfDay(new Date());
  return dueDate.getTime() < today.getTime();
}

/**
 * Check if a date is due today
 */
export function isDueToday(date: string | null): boolean {
  if (!date) return false;
  const dueDate = startOfDay(new Date(date));
  const today = startOfDay(new Date());
  return dueDate.getTime() === today.getTime();
}

/**
 * Check if a date is due this week
 */
export function isDueThisWeek(date: string | null): boolean {
  if (!date) return false;
  const dueDate = new Date(date);
  const today = startOfDay(new Date());
  const diffDays = Math.round((dueDate.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
  return diffDays >= 0 && diffDays <= 7;
}

/**
 * Convert Date to datetime-local input format
 */
export function dateToInputFormat(date: Date): string {
  return format(date, "yyyy-MM-dd'T'HH:mm");
}

/**
 * Get segments of text with date portions highlighted
 */
export interface TextSegment {
  text: string;
  isDate: boolean;
  date?: Date;
}

export function getTextSegments(text: string): TextSegment[] {
  const parsed = parseDateFromText(text);
  if (!parsed) {
    return [{ text, isDate: false }];
  }

  const segments: TextSegment[] = [];

  if (parsed.startIndex > 0) {
    segments.push({ text: text.substring(0, parsed.startIndex), isDate: false });
  }

  segments.push({
    text: parsed.matchedText,
    isDate: true,
    date: parsed.date ?? undefined
  });

  if (parsed.endIndex < text.length) {
    segments.push({ text: text.substring(parsed.endIndex), isDate: false });
  }

  return segments;
}
