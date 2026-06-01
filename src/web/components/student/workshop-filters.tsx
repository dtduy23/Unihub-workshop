"use client"

import { Search } from "lucide-react"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const categories = [
  { value: "all", label: "Tất cả chủ đề" },
  { value: "tech", label: "Công nghệ" },
  { value: "design", label: "Thiết kế" },
  { value: "business", label: "Kinh doanh" },
  { value: "marketing", label: "Marketing" },
  { value: "softskill", label: "Kỹ năng mềm" },
]

const ticketTypes = [
  { value: "all", label: "Tất cả loại vé" },
  { value: "free", label: "Miễn phí" },
  { value: "paid", label: "Có phí" },
  { value: "registered", label: "Đã đăng ký" },
]

const dateFilters = [
  { value: "all", label: "Tất cả thời gian" },
  { value: "today", label: "Hôm nay" },
  { value: "this-week", label: "Tuần này" },
  { value: "this-month", label: "Tháng này" },
]

interface WorkshopFiltersProps {
  onSearch: (value: string) => void
  onCategoryChange: (value: string) => void
  onTicketTypeChange: (value: string) => void
}

export function WorkshopFilters({ 
  onSearch, 
  onCategoryChange, 
  onTicketTypeChange 
}: WorkshopFiltersProps) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
      {/* Search Input */}
      <div className="relative flex-1">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          type="search"
          placeholder="Tìm kiếm workshop..."
          className="h-11 pl-10 bg-background border-slate-200 focus:border-primary transition-all"
          onChange={(e) => onSearch(e.target.value)}
        />
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-2 sm:flex-nowrap">
        <Select defaultValue="all" onValueChange={onCategoryChange}>
          <SelectTrigger className="h-11 w-full sm:w-[160px] bg-background">
            <SelectValue placeholder="Chủ đề" />
          </SelectTrigger>
          <SelectContent>
            {categories.map((category) => (
              <SelectItem key={category.value} value={category.value}>
                {category.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select defaultValue="all" onValueChange={onTicketTypeChange}>
          <SelectTrigger className="h-11 w-full sm:w-[140px] bg-background">
            <SelectValue placeholder="Loại vé" />
          </SelectTrigger>
          <SelectContent>
            {ticketTypes.map((type) => (
              <SelectItem key={type.value} value={type.value}>
                {type.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        
      </div>
    </div>
  )
}
