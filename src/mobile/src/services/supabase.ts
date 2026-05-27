import 'react-native-url-polyfill/auto';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { createClient } from '@supabase/supabase-js';

const supabaseUrl = 'https://hmgrbkzdstqcfhznvrqd.supabase.co';
const supabaseAnonKey = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImhtZ3Jia3pkc3RxY2Zoem52cnFkIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NzgzMjE2MzUsImV4cCI6MjA5Mzg5NzYzNX0.eKSZ7hhTF1EhGs-ljHiRK38NqI58W4GsjuZAAMs44pw';

export const supabase = createClient(supabaseUrl, supabaseAnonKey, {
  auth: {
    storage: AsyncStorage,
    autoRefreshToken: true,
    persistSession: true,
    detectSessionInUrl: false,
  },
});
