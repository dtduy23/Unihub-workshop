"use client"

import { useState } from "react"
import { Navbar } from "@/components/student/navbar"
import { WorkshopFilters } from "@/components/student/workshop-filters"
import { WorkshopGrid } from "@/components/student/workshop-grid"

export default function HomePage() {
  const [search, setSearch] = useState("")
  const [category, setCategory] = useState("all")
  const [ticketType, setTicketType] = useState("all")

  return (
    <div className="min-h-screen bg-background">
      <Navbar />
      
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Page Header */}
        <div className="mb-8 text-center sm:text-left">
          <h1 className="text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Khám phá Workshop
          </h1>
          <p className="mt-2 text-lg text-muted-foreground">
            Tìm và đăng ký các workshop hấp dẫn nhất tại UniHub
          </p>
        </div>

        {/* Filters Section */}
        <div className="mb-10">
          <WorkshopFilters 
            onSearch={setSearch}
            onCategoryChange={setCategory}
            onTicketTypeChange={setTicketType}
          />
        </div>

        {/* Workshop Grid */}
        <WorkshopGrid 
          searchQuery={search}
          categoryFilter={category}
          ticketTypeFilter={ticketType}
        />
      </main>
    </div>
  )
}
