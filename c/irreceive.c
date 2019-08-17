#include <stdio.h>

#include <pigpio.h>

#define IR_PIN 25

#define OUTSIDE_CODE 0
#define INSIDE_CODE 1

#define MIN_MESSAGE_GAP 3000
#define MAX_MESSAGE_END 3000

#define MAX_TRANSITIONS 500

/*
   using the FNV-1a hash                 
   from http://isthe.com/chongo/tech/comp/fnv/#FNV-param
*/

#define FNV_PRIME_32 16777619
#define FNV_BASIS_32 2166136261U

static volatile uint32_t ir_hash = 0;

typedef struct
{
   int state;
   int count;
   int level;
   uint16_t micros[MAX_TRANSITIONS];
} decode_t;

/* forward declarations */

void alert(int gpio, int level, uint32_t tick);
uint32_t getHash(decode_t *decode);
void updateState(decode_t *decode, int level, uint32_t micros);

int main(int argc, char *argv[])
{
   printf("start\n");

   if (gpioInitialise() < 0)
   {
      return 1;
   }

   /* IR pin as input */

   gpioSetMode(IR_PIN, PI_INPUT);

   /* 5ms max gap after last pulse */

   gpioSetWatchdog(IR_PIN, 5);

   /* monitor IR level changes */

   gpioSetAlertFunc(IR_PIN, alert);

   while (1)
   {
      if (ir_hash)
      {
         /* non-zero means new decode */
         //printf("ir hash is %u\n", ir_hash);
         ir_hash = 0;
         printf("end\n");
         return 0;
      }

      gpioDelay(100000); /* check remote 10 times per second */
   }

   gpioTerminate();
}

void alert(int gpio, int level, uint32_t tick)
{
   static int inited = 0;

   static decode_t activeHigh, activeLow;

   static uint32_t lastTick;

   uint32_t diffTick;

   if (!inited)
   {
      inited = 1;

      activeHigh.state = OUTSIDE_CODE;
      activeHigh.level = PI_LOW;
      activeLow.state = OUTSIDE_CODE;
      activeLow.level = PI_HIGH;

      lastTick = tick;
      return;
   }

   diffTick = tick - lastTick;
   if (level != 2)
   {
      printf("%d %u\n", level, diffTick);
   }

   if (level != PI_TIMEOUT)
      lastTick = tick;

   updateState(&activeHigh, level, diffTick);
   updateState(&activeLow, level, diffTick);
}

void updateState(decode_t *decode, int level, uint32_t micros)
{
   /*
      We are dealing with active high as well as active low
      remotes.  Abstract the common functionality.
   */

   if (decode->state == OUTSIDE_CODE)
   {
      if (level == decode->level)
      {
         if (micros > MIN_MESSAGE_GAP)
         {
            decode->state = INSIDE_CODE;
            decode->count = 0;
         }
      }
   }
   else
   {
      if (micros > MAX_MESSAGE_END)
      {
         /* end of message */

         /* ignore if last code not consumed */

         if (!ir_hash)
            ir_hash = getHash(decode);

         decode->state = OUTSIDE_CODE;
      }
      else
      {
         if (decode->count < (MAX_TRANSITIONS - 1))
         {
            if (level != PI_TIMEOUT)
               decode->micros[decode->count++] = micros;
         }
      }
   }
}

int compare(unsigned int oldval, unsigned int newval)
{
   if (newval < (oldval * 0.75))
   {
      return 1;
   }
   else if (oldval < (newval * 0.75))
   {
      return 2;
   }
   else
   {
      return 4;
   }
}

uint32_t getHash(decode_t *decode)
{
   /* use FNV-1a */

   uint32_t hash;
   int i, value;

   if (decode->count < 6)
   {
      return 0;
   }

   hash = FNV_BASIS_32;

   for (i = 0; i < (decode->count - 2); i++)
   {
      value = compare(decode->micros[i], decode->micros[i + 2]);

      hash = hash ^ value;
      hash = (hash * FNV_PRIME_32);
   }

   return hash;
}
