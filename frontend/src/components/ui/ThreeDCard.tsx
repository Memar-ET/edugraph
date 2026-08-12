import React, { useRef, useState } from 'react'
import { cn } from '@lib/utils/cn'

export interface ThreeDCardProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode
  className?: string
  depth?: number
  glare?: boolean
}

export function ThreeDCard({
  children,
  className,
  depth = 15,
  glare = true,
  ...props
}: ThreeDCardProps) {
  const cardRef = useRef<HTMLDivElement>(null)
  const [rotateX, setRotateX] = useState(0)
  const [rotateY, setRotateY] = useState(0)
  const [glarePos, setGlarePos] = useState({ x: 50, y: 50, opacity: 0 })
  const [isHovered, setIsHovered] = useState(false)

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!cardRef.current) return
    const rect = cardRef.current.getBoundingClientRect()
    const width = rect.width
    const height = rect.height

    const mouseX = e.clientX - rect.left
    const mouseY = e.clientY - rect.top

    const rX = ((mouseY - height / 2) / (height / 2)) * -depth
    const rY = ((mouseX - width / 2) / (width / 2)) * depth

    setRotateX(rX)
    setRotateY(rY)

    const glareX = (mouseX / width) * 100
    const glareY = (mouseY / height) * 100
    setGlarePos({ x: glareX, y: glareY, opacity: 0.15 })
  }

  const handleMouseEnter = () => {
    setIsHovered(true)
  }

  const handleMouseLeave = () => {
    setIsHovered(false)
    setRotateX(0)
    setRotateY(0)
    setGlarePos((prev) => ({ ...prev, opacity: 0 }))
  }

  return (
    <div
      className="perspective-1000"
      style={{ perspective: '1000px' }}
    >
      <div
        ref={cardRef}
        onMouseMove={handleMouseMove}
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        style={{
          transform: isHovered
            ? `rotateX(${rotateX}deg) rotateY(${rotateY}deg) translateZ(8px)`
            : 'rotateX(0deg) rotateY(0deg) translateZ(0px)',
          transition: isHovered ? 'transform 0.1s cubic-bezier(0.03, 0.98, 0.52, 0.99)' : 'transform 0.5s ease-out',
          transformStyle: 'preserve-3d',
        }}
        className={cn(
          'relative overflow-hidden rounded-2xl border border-gray-100/90 bg-white p-5 shadow-[0_4px_20px_-2px_rgba(0,0,0,0.05)] transition-all duration-300 hover:shadow-[0_20px_35px_-5px_rgba(0,0,0,0.08)]',
          className,
        )}
        {...props}
      >
        {/* Interactive 3D Glare Reflection Effect */}
        {glare && (
          <div
            className="pointer-events-none absolute inset-0 z-10 rounded-2xl transition-opacity duration-300"
            style={{
              background: `radial-gradient(circle at ${glarePos.x}% ${glarePos.y}%, rgba(255,255,255,0.8) 0%, rgba(255,255,255,0) 60%)`,
              opacity: glarePos.opacity,
            }}
          />
        )}

        {/* 3D Depth Layer */}
        <div style={{ transform: 'translateZ(12px)', transformStyle: 'preserve-3d' }}>
          {children}
        </div>
      </div>
    </div>
  )
}
