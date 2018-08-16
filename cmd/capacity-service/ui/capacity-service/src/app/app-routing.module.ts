import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Routes } from '@angular/router';

import { CapacityServiceComponent } from "./capacity-service/capacity-service.component";

const routes: Routes = [
	{ path: '', component: CapacityServiceComponent }
]

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule],
})
export class AppRoutingModule { }
