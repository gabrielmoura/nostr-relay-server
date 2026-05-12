import React, { useState, useEffect, useCallback } from "react"
import useEmblaCarousel from "embla-carousel-react"
import { ChevronLeft, ChevronRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { EventVideoPlayer } from "./event-video-player"

export type MediaItem = {
  type: "image" | "video"
  url: string
  alt?: string
}

interface MediaCarouselProps {
  media: MediaItem[]
  poster?: string
  className?: string
  viewportClassName?: string
  mediaClassName?: string
  lazyVideo?: boolean
}

export function MediaCarousel({ media, poster, className, viewportClassName, mediaClassName, lazyVideo = false }: MediaCarouselProps) {
  const [emblaRef, emblaApi] = useEmblaCarousel({ loop: true })
  const [prevBtnDisabled, setPrevBtnDisabled] = useState(true)
  const [nextBtnDisabled, setNextBtnDisabled] = useState(true)
  const [selectedIndex, setSelectedIndex] = useState(0)

  const scrollPrev = useCallback(() => emblaApi && emblaApi.scrollPrev(), [emblaApi])
  const scrollNext = useCallback(() => emblaApi && emblaApi.scrollNext(), [emblaApi])
  const scrollTo = useCallback((index: number) => emblaApi && emblaApi.scrollTo(index), [emblaApi])

  const onSelect = useCallback(() => {
    if (!emblaApi) return
    setSelectedIndex(emblaApi.selectedScrollSnap())
    setPrevBtnDisabled(!emblaApi.canScrollPrev())
    setNextBtnDisabled(!emblaApi.canScrollNext())
  }, [emblaApi])

  useEffect(() => {
    if (!emblaApi) return
    onSelect()
    emblaApi.on("select", onSelect)
    emblaApi.on("reInit", onSelect)
  }, [emblaApi, onSelect])

  return (
    <div className={cn("relative group overflow-hidden rounded-md border border-border bg-black", className)}>
      <div className={cn("overflow-hidden", viewportClassName)} ref={emblaRef}>
        <div className="flex touch-pan-y">
          {media.map((item, index) => (
            <div className="min-w-0 flex-[0_0_100%] flex items-center justify-center relative bg-black" key={`${item.type}-${item.url}-${index}`}>
              {item.type === "video" ? (
                <EventVideoPlayer
                  className={cn("max-h-[60vh] w-full rounded-none border-0 bg-black object-contain p-0", mediaClassName)}
                  lazy={lazyVideo}
                  poster={index === 0 ? poster : undefined}
                  showChrome={false}
                  url={item.url}
                />
              ) : (
                <img
                  className={cn("max-h-[60vh] w-full object-contain", mediaClassName)}
                  src={item.url}
                  alt={item.alt || `Media ${index + 1}`}
                  loading={index === 0 ? "eager" : "lazy"}
                />
              )}
            </div>
          ))}
        </div>
      </div>

      {media.length > 1 && (
        <>
          <div className="absolute inset-y-0 left-2 flex items-center">
            <Button
              variant="secondary"
              size="icon"
              className={cn(
                "h-8 w-8 rounded-full bg-background/50 backdrop-blur-sm opacity-0 group-hover:opacity-100 transition-opacity hover:bg-background/80",
                prevBtnDisabled && "hidden"
              )}
              onClick={scrollPrev}
              disabled={prevBtnDisabled}
            >
              <ChevronLeft className="h-4 w-4" />
              <span className="sr-only">Anterior</span>
            </Button>
          </div>
          
          <div className="absolute inset-y-0 right-2 flex items-center">
            <Button
              variant="secondary"
              size="icon"
              className={cn(
                "h-8 w-8 rounded-full bg-background/50 backdrop-blur-sm opacity-0 group-hover:opacity-100 transition-opacity hover:bg-background/80",
                nextBtnDisabled && "hidden"
              )}
              onClick={scrollNext}
              disabled={nextBtnDisabled}
            >
              <ChevronRight className="h-4 w-4" />
              <span className="sr-only">Próximo</span>
            </Button>
          </div>

          <div className="absolute bottom-4 left-0 right-0 flex justify-center gap-2">
            {media.map((_, index) => (
              <button
                key={index}
                className={cn(
                  "h-1.5 rounded-full transition-all duration-300",
                  index === selectedIndex 
                    ? "w-4 bg-primary" 
                    : "w-1.5 bg-primary/30 hover:bg-primary/50"
                )}
                onClick={() => scrollTo(index)}
                aria-label={`Ir para slide ${index + 1}`}
              />
            ))}
          </div>
        </>
      )}
    </div>
  )
}
