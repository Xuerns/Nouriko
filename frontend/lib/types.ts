export type Language = "id" | "en";
export type Theme = "light" | "dark";
export type Gender = "male" | "female";
export type ActivityLevel = "light" | "moderate" | "heavy";

export interface ProfileDTO {
  id: string;
  email: string;
  profile_url: string;
  display_name: string;
  height_cm: number | null;
  weight_kg: number | null;
  age: number | null;
  gender: Gender | null;
  activity_level: ActivityLevel | null;
  daily_calorie_target: number | null;
  target_protein_g: number | null;
  target_carbs_g: number | null;
  target_fat_g: number | null;
  target_fiber_g: number | null;
  language: Language;
  theme: Theme;
  onboarding_completed: boolean;
}

export interface RegisterResponse {
  status: string;
  user_id: string;
  email: string;
  dev_verification_token?: string;
}

export interface LoginResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  profile: ProfileDTO;
}

export interface StatusResponse {
  status: string;
  dev_reset_token?: string;
}

export interface ProfileResponse {
  profile: ProfileDTO;
}

export interface OnboardingPayload {
  display_name?: string;
  height_cm: number;
  weight_kg: number;
  age: number;
  gender: Gender;
  activity_level: ActivityLevel;
}

export interface UpdateProfilePayload {
  display_name?: string;
  height_cm?: number;
  weight_kg?: number;
  age?: number;
  gender?: Gender;
  activity_level?: ActivityLevel;
  daily_calorie_target?: number;
  target_protein_g?: number;
  target_carbs_g?: number;
  target_fat_g?: number;
  target_fiber_g?: number;
  language?: Language;
  theme?: Theme;
}

export type ChatIntent =
  | "ingredients"
  | "budget_meals"
  | "budget_ingredients"
  | "photo_food"
  | "photo_ingredients"
  | "off_topic";

export type HealthLabel = "low" | "fair" | "good";
export type InputSource = "ingredients" | "budget" | "photo";
export type MacroSource = "food_api" | "estimated";

export interface ConversationTurn {
  role: "user" | "assistant";
  summary: string;
}

export interface MealOption {
  name: string;
  portion: string;
  health_score: number;
  health_label: HealthLabel;
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
  estimated_price_idr: number;
  is_price_estimated: boolean;
  is_macro_estimated: boolean;
  macro_source: MacroSource;
  note: string;
  cooking_steps: string[];
  missing_ingredients: string[];
  input_source: InputSource;
}

export interface ShoppingListItem {
  name: string;
  unit: string;
  estimated_price_idr: number;
  is_price_estimated: boolean;
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
}

export interface ChatAnalyzeResponse {
  intent: ChatIntent;
  intro: string;
  off_topic_notice?: string;
  detected_items: string[];
  needs_clarification: boolean;
  budget_idr?: number;
  budget_exceeded_notice?: string;
  meals: MealOption[];
  shopping_list: ShoppingListItem[];
}

export interface AnalyzeChatParams {
  message?: string;
  photo?: File | null;
  language: Language;
  conversationContext?: ConversationTurn[];
}

export interface ChatMessage {
  id: string;
  role: "user" | "assistant";
  text: string;
  photoPreviewUrl?: string;
  photoName?: string;
  response?: ChatAnalyzeResponse;
  errorCode?: string;
  hasPhoto?: boolean;
  createdAt: number;
}

export type MealTime = "breakfast" | "lunch" | "dinner" | "snack";

export interface SavedMealDTO {
  id: string;
  meal_date: string;
  meal_time: MealTime;
  name: string;
  health_score: number;
  health_label: HealthLabel;
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
  price_idr: number | null;
  input_source: InputSource;
  created_at: string;
}

export interface CreateSavedMealRequest {
  meal_date?: string;
  meal_time?: MealTime;
  name: string;
  health_score: number;
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
  price_idr?: number | null;
  input_source: InputSource;
}

export interface CreateSavedMealResponse {
  meal: SavedMealDTO;
}

export interface DeleteSavedMealResponse {
  deleted: boolean;
}

export interface DailyTotals {
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
  total_spending_idr: number;
  meal_count: number;
  average_health_score: number;
  health_label: HealthLabel | null;
}

export interface DailyTargets {
  daily_calorie_target: number;
  target_protein_g: number;
  target_carbs_g: number;
  target_fat_g: number;
  target_fiber_g: number;
}

export interface DailyProgressPercent {
  calories: number;
  protein_g: number;
  carbs_g: number;
  fat_g: number;
  fiber_g: number;
}

export interface DailySummaryResponse {
  date: string;
  meals: SavedMealDTO[];
  totals: DailyTotals;
  targets: DailyTargets;
  progress_percent: DailyProgressPercent;
}

export interface HistoryDaySummary {
  date: string;
  meal_count: number;
  total_calories: number;
  total_protein_g: number;
  total_carbs_g: number;
  total_fat_g: number;
  total_fiber_g: number;
  total_spending_idr: number;
  average_health_score: number;
  health_label: HealthLabel;
  meals: SavedMealDTO[];
}

export interface WeeklyHealthTrendPoint {
  date: string;
  average_health_score: number | null;
  meal_count: number;
}

export interface MealHistoryResponse {
  from: string;
  to: string;
  days: HistoryDaySummary[];
  weekly_trend: WeeklyHealthTrendPoint[];
}
