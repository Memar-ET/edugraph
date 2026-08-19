import { useState } from 'react'
import { MapPin } from 'lucide-react'

interface RegionData {
  id: string
  name: string
  score: number
  schoolsCount: number
  color: string
}

const REGIONS: RegionData[] = [
  { id: 'AA', name: 'Addis Ababa', score: 84.2, schoolsCount: 142, color: '#059669' },
  { id: 'OR', name: 'Oromia', score: 76.5, schoolsCount: 890, color: '#0d9488' },
  { id: 'AM', name: 'Amhara', score: 74.8, schoolsCount: 650, color: '#0d9488' },
  { id: 'SD', name: 'Sidama', score: 68.2, schoolsCount: 220, color: '#d97706' },
  { id: 'TG', name: 'Tigray', score: 72.0, schoolsCount: 310, color: '#0d9488' },
  { id: 'SO', name: 'Somali', score: 61.4, schoolsCount: 180, color: '#dc2626' },
  { id: 'DD', name: 'Dire Dawa', score: 79.1, schoolsCount: 45, color: '#059669' },
  { id: 'AF', name: 'Afar', score: 64.0, schoolsCount: 95, color: '#d97706' },
  { id: 'BG', name: 'Benishangul', score: 67.5, schoolsCount: 80, color: '#d97706' },
  { id: 'GM', name: 'Gambela', score: 63.8, schoolsCount: 60, color: '#dc2626' },
]

export function ChoroplethMap({ onRegionSelect }: { onRegionSelect?: (regionId: string) => void }) {
  const [hoveredRegion, setHoveredRegion] = useState<RegionData | null>(REGIONS[0]!)

  return (
    <div className="relative rounded-2xl border border-slate-200 bg-slate-900 p-6 text-white min-h-[380px] overflow-hidden">
      {/* Header overlay */}
      <div className="flex items-center justify-between border-b border-slate-800 pb-4 mb-4">
        <div>
          <div className="flex items-center gap-2">
            <MapPin className="h-4 w-4 text-teal-400" />
            <h3 className="font-bold text-sm text-white">Ethiopian National Equity & Mastery Map</h3>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">Click any region to inspect district school scorecards</p>
        </div>
        {hoveredRegion && (
          <div className="rounded-xl border border-slate-700 bg-slate-800/90 px-3 py-1.5 text-xs text-right backdrop-blur-md">
            <p className="font-bold text-teal-300">{hoveredRegion.name}</p>
            <p className="font-mono text-[11px] text-slate-300">
              Mastery: <strong className="text-white">{hoveredRegion.score}%</strong> ({hoveredRegion.schoolsCount} schools)
            </p>
          </div>
        )}
      </div>

      {/* SVG Map Grid Layout */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5 py-2">
        {REGIONS.map((region) => {
          const isSelected = hoveredRegion?.id === region.id
          return (
            <button
              key={region.id}
              type="button"
              onMouseEnter={() => setHoveredRegion(region)}
              onClick={() => onRegionSelect && onRegionSelect(region.id)}
              className={`flex flex-col justify-between rounded-xl border p-3 text-left transition-all ${
                isSelected
                  ? 'border-teal-400 bg-slate-800 ring-2 ring-teal-500/50 shadow-lg scale-105'
                  : 'border-slate-800 bg-slate-800/60 hover:border-slate-600'
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="font-mono text-xs font-bold text-slate-400">{region.id}</span>
                <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: region.color }} />
              </div>
              <p className="mt-2 font-bold text-xs text-white truncate">{region.name}</p>
              <div className="mt-2 flex items-center justify-between text-[10px] text-slate-400 font-mono">
                <span>{region.schoolsCount} Schools</span>
                <span className="font-bold text-teal-400">{region.score}%</span>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
